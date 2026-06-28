package chat

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/docaccess"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/document"
)

// ChatStream runs the 4-phase RAG pipeline with SSE streaming (pattern from AI-Hukum-BE).
func (uc *ChatUsecase) ChatStream(
	ctx context.Context,
	access docaccess.Context,
	conversationID string,
	message string,
	documentID string,
	chatMode document.ChatMode,
) (<-chan dto.StreamChunk, error) {
	ch := make(chan dto.StreamChunk, 32)
	userID := access.UserID

	go func() {
		defer close(ch)
		start := time.Now()

		message = strings.TrimSpace(message)
		if message == "" {
			ch <- dto.StreamChunk{Type: "error", Error: "message is required"}
			return
		}

		var conv *entity.Conversation
		var history []entity.Message

		if conversationID != "" {
			c, err := uc.convRepo.FindByIDAndUserID(ctx, conversationID, userID)
			if err != nil || c == nil {
				ch <- dto.StreamChunk{Type: "error", Error: "conversation not found"}
				return
			}
			conv = c
			history, _ = uc.msgRepo.ListByConversation(ctx, conversationID, 20)
		} else {
			conv = &entity.Conversation{UserID: userID, Title: generateTitle(message)}
			if scope := strings.TrimSpace(documentID); scope != "" {
				conv.DocumentID = &scope
				conv.Title = uc.documentScopedTitle(ctx, scope, conv.Title)
			}
			if err := uc.convRepo.Create(ctx, conv); err != nil {
				ch <- dto.StreamChunk{Type: "error", Error: "failed to create conversation"}
				return
			}
		}

		ch <- dto.StreamChunk{Type: "conversation_id", ConversationID: conv.ID}

		userMsg := &entity.Message{
			ConversationID: conv.ID,
			Role:           entity.MessageRoleUser,
			Content:        message,
		}
		if err := uc.msgRepo.Create(ctx, userMsg); err != nil {
			ch <- dto.StreamChunk{Type: "error", Error: "failed to save user message"}
			return
		}

		chatHistory := toChatHistory(history)
		rag, err := uc.docUC.PrepareRAGForDocument(ctx, access, conversationDocumentScope(conv), message, chatHistory)
		if err != nil {
			ch <- dto.StreamChunk{Type: "error", Error: err.Error()}
			return
		}

		if rag.Answer != "" {
			ch <- dto.StreamChunk{Type: "content", Content: rag.Answer}
			uc.saveAssistantAndLog(ctx, conv, message, rag.Answer, nil, rag, start, userID)
			ch <- dto.StreamChunk{Type: "done", Metadata: map[string]any{
				"responseTimeMs":   time.Since(start).Milliseconds(),
				"chatMode":         string(chatMode),
				"responseStrategy": "greeting",
				"contextUsed":      false,
			}}
			return
		}

		plan := document.PlanRAGResponse(chatMode, rag, message)

		sources := buildChunkSources(ctx, rag.DisplaySources, uc.docUC)
		if len(sources) > 0 {
			ch <- dto.StreamChunk{Type: "sources", Sources: sources}
		}

		var answer string
		if !plan.UseLLM {
			answer = plan.DirectAnswer
			ch <- dto.StreamChunk{Type: "content", Content: answer}
		} else {
			streamCh, errCh := uc.docUC.GenerateAnswerStream(ctx, message, plan.DocContext, chatHistory)

			var answerBuilder strings.Builder
			for token := range streamCh {
				answerBuilder.WriteString(token)
				ch <- dto.StreamChunk{Type: "content", Content: token}
			}

			if streamErr := <-errCh; streamErr != nil {
				ch <- dto.StreamChunk{Type: "error", Error: streamErr.Error()}
				return
			}
			answer = answerBuilder.String()
		}

		uc.saveAssistantAndLog(ctx, conv, message, answer, sources, rag, start, userID)

		if conv.Title == "Chat Baru" || conv.Title == "" {
			conv.Title = generateTitle(message)
			_ = uc.convRepo.Update(ctx, conv)
		}

		ch <- dto.StreamChunk{Type: "done", Metadata: map[string]any{
			"responseTimeMs":   time.Since(start).Milliseconds(),
			"searchType":       rag.SearchType,
			"chatMode":         string(chatMode),
			"responseStrategy": string(plan.Strategy),
			"contextUsed":      plan.Strategy == document.ResponseFromDocuments && len(sources) > 0,
		}}
	}()

	return ch, nil
}

func (uc *ChatUsecase) saveAssistantAndLog(
	ctx context.Context,
	conv *entity.Conversation,
	query, answer string,
	sources []dto.ChunkSource,
	rag *document.RAGResult,
	start time.Time,
	userID string,
) {
	sourcesJSON := buildSourcesJSONFromDTO(sources)
	assistantMsg := &entity.Message{
		ConversationID: conv.ID,
		Role:           entity.MessageRoleAssistant,
		Content:        answer,
		Sources:        sourcesJSON,
	}
	_ = uc.msgRepo.Create(ctx, assistantMsg)

	if uc.queryLogRepo != nil && rag != nil {
		convID := conv.ID
		var reformulated *string
		if rag.ReformulatedQuery != "" {
			reformulated = &rag.ReformulatedQuery
		}
		searchType := rag.SearchType
		if searchType == "" {
			searchType = "vector"
		}
		_ = uc.queryLogRepo.Create(ctx, &entity.QueryLog{
			ConversationID:    &convID,
			UserID:            userID,
			Query:             query,
			ReformulatedQuery: reformulated,
			SearchType:        searchType,
			ChunksRetrieved:   len(rag.Chunks),
			ResponseTimeMs:    int(time.Since(start).Milliseconds()),
		})
	}
}

func buildChunkSources(ctx context.Context, chunks []entity.SimilarChunk, docUC DocumentQuerier) []dto.ChunkSource {
	sources := make([]dto.ChunkSource, 0, len(chunks))
	for _, chunk := range chunks {
		source := dto.ChunkSource{
			DocumentID: chunk.DocumentID,
			Similarity: chunk.Similarity,
			Content:    chunk.Content,
			ChunkIndex: chunk.ChunkIndex,
		}
		if name := docUC.GetDocumentOriginalName(ctx, chunk.DocumentID); name != "" {
			source.DocumentName = name
		}
		if len(chunk.Metadata) > 0 {
			var meta entity.ChunkMetadata
			if json.Unmarshal(chunk.Metadata, &meta) == nil {
				source.PageNumber = meta.PageNumber
			}
		}
		source.LowConfidence = document.IsLowConfidenceSource(source.Similarity, source.Content)
		sources = append(sources, source)
	}
	return sources
}

func buildSourcesJSONFromDTO(sources []dto.ChunkSource) []byte {
	if len(sources) == 0 {
		return nil
	}
	data, _ := json.Marshal(sources)
	return data
}
