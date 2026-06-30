package chat

import (
	"context"
	"encoding/json"
	"log"
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
		ch <- dto.StreamChunk{Type: "status", Content: "Mencari konteks dokumen..."}

		userMsg := &entity.Message{
			ConversationID: conv.ID,
			Role:           entity.MessageRoleUser,
			Content:        message,
		}
		saveCtx := context.WithoutCancel(ctx)
		go func() {
			if err := uc.msgRepo.Create(saveCtx, userMsg); err != nil {
				log.Printf("[chat/stream] failed to save user message: %v", err)
			}
		}()

		chatHistory := toChatHistory(history)

		type ragOutcome struct {
			result *document.RAGResult
			err    error
		}
		ragCh := make(chan ragOutcome, 1)
		go func() {
			result, err := uc.docUC.PrepareRAGForDocument(
				ctx,
				access,
				conversationDocumentScope(conv),
				message,
				chatHistory,
			)
			ragCh <- ragOutcome{result: result, err: err}
		}()

		heartbeat := time.NewTicker(2 * time.Second)
		defer heartbeat.Stop()

		var rag *document.RAGResult
		var ragErr error
		waitingRAG := true
		for waitingRAG {
			select {
			case <-heartbeat.C:
				ch <- dto.StreamChunk{Type: "status", Content: "Mencari konteks dokumen..."}
			case outcome := <-ragCh:
				rag = outcome.result
				ragErr = outcome.err
				waitingRAG = false
			case <-ctx.Done():
				return
			}
		}

		if ragErr != nil {
			ch <- dto.StreamChunk{Type: "error", Error: ragErr.Error()}
			return
		}

		if rag.Answer != "" {
			streamTextChunks(ch, rag.Answer)
			ch <- dto.StreamChunk{Type: "done", Metadata: map[string]any{
				"responseTimeMs":   time.Since(start).Milliseconds(),
				"chatMode":         string(chatMode),
				"responseStrategy": "greeting",
				"contextUsed":      false,
			}}
			saveCtx = context.WithoutCancel(ctx)
			go func() {
				if err := uc.saveAssistantAndLog(saveCtx, conv, message, rag.Answer, nil, rag, start, userID); err != nil {
					log.Printf("[chat/stream] failed to save greeting response: %v", err)
				}
			}()
			return
		}

		plan := document.PlanRAGResponse(chatMode, rag, message)
		sources := buildChunkSources(rag.DisplaySources, uc.docUC)
		if len(sources) > 0 {
			ch <- dto.StreamChunk{Type: "sources", Sources: sources}
		}

		var answer string
		if !plan.UseLLM {
			answer = plan.DirectAnswer
			streamTextChunks(ch, answer)
		} else {
			ch <- dto.StreamChunk{Type: "status", Content: "Menyusun jawaban..."}

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

		saveCtx = context.WithoutCancel(ctx)
		go func() {
			if err := uc.saveAssistantAndLog(saveCtx, conv, message, answer, sources, rag, start, userID); err != nil {
				log.Printf("[chat/stream] failed to save assistant response: %v", err)
			}
		}()
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
) error {
	sourcesJSON := buildSourcesJSONFromDTO(sources)
	assistantMsg := &entity.Message{
		ConversationID: conv.ID,
		Role:           entity.MessageRoleAssistant,
		Content:        answer,
		Sources:        sourcesJSON,
	}
	if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
		return err
	}

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

	return nil
}

func streamTextChunks(ch chan<- dto.StreamChunk, text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	parts := strings.Fields(text)
	if len(parts) == 0 {
		ch <- dto.StreamChunk{Type: "content", Content: text}
		return
	}

	for i, part := range parts {
		token := part
		if i < len(parts)-1 {
			token += " "
		}
		ch <- dto.StreamChunk{Type: "content", Content: token}
	}
}

func buildChunkSources(chunks []entity.SimilarChunk, docUC DocumentQuerier) []dto.ChunkSource {
	sources := make([]dto.ChunkSource, 0, len(chunks))
	for _, chunk := range chunks {
		source := dto.ChunkSource{
			DocumentID: chunk.DocumentID,
			Similarity: chunk.Similarity,
			Content:    chunk.Content,
			ChunkIndex: chunk.ChunkIndex,
		}
		name := strings.TrimSpace(chunk.DocumentName)
		if name == "" && docUC != nil {
			name = docUC.GetDocumentOriginalName(context.Background(), chunk.DocumentID)
		}
		if name != "" {
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
