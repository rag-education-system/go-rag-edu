package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
	"rag-api/internal/usecase/document"
)

type DocumentQuerier interface {
	QueryDocuments(ctx context.Context, userID, query string, history []document.ChatMessage) (string, []entity.SimilarChunk, error)
	PrepareRAG(ctx context.Context, userID, query string, history []document.ChatMessage) (*document.RAGResult, error)
	GenerateAnswerStream(ctx context.Context, query, docContext string, history []document.ChatMessage) (<-chan string, <-chan error)
	GetDocumentOriginalName(ctx context.Context, documentID string) string
}

type ChatUsecase struct {
	convRepo     repository.ConversationRepository
	msgRepo      repository.MessageRepository
	queryLogRepo repository.QueryLogRepository
	docUC        DocumentQuerier
}

func NewChatUsecase(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	queryLogRepo repository.QueryLogRepository,
	docUC DocumentQuerier,
) *ChatUsecase {
	return &ChatUsecase{
		convRepo:     convRepo,
		msgRepo:      msgRepo,
		queryLogRepo: queryLogRepo,
		docUC:        docUC,
	}
}

func (uc *ChatUsecase) CreateConversation(
	ctx context.Context,
	userID, message string,
) (*entity.Conversation, *entity.Message, *entity.Message, error) {
	title := generateTitle(message)
	conv := &entity.Conversation{
		UserID: userID,
		Title:  title,
	}

	if err := uc.convRepo.Create(ctx, conv); err != nil {
		return nil, nil, nil, err
	}

	userMsg, assistantMsg, err := uc.processMessage(ctx, conv, message, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	return conv, userMsg, assistantMsg, nil
}

func (uc *ChatUsecase) SendMessage(
	ctx context.Context,
	conversationID, userID, message string,
) (*entity.Message, *entity.Message, error) {
	conv, err := uc.convRepo.FindByIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, err
	}
	if conv == nil {
		return nil, nil, fmt.Errorf("conversation not found")
	}

	history, err := uc.msgRepo.ListByConversation(ctx, conversationID, 20)
	if err != nil {
		return nil, nil, err
	}

	return uc.processMessage(ctx, conv, message, history)
}

func (uc *ChatUsecase) processMessage(
	ctx context.Context,
	conv *entity.Conversation,
	message string,
	history []entity.Message,
) (*entity.Message, *entity.Message, error) {
	userMsg := &entity.Message{
		ConversationID: conv.ID,
		Role:           entity.MessageRoleUser,
		Content:        strings.TrimSpace(message),
	}
	if err := uc.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, nil, err
	}

	chatHistory := toChatHistory(history)
	answer, chunks, err := uc.docUC.QueryDocuments(ctx, conv.UserID, message, chatHistory)
	if err != nil {
		return nil, nil, err
	}

	sourcesJSON := buildSourcesJSON(ctx, chunks, uc.docUC)
	assistantMsg := &entity.Message{
		ConversationID: conv.ID,
		Role:           entity.MessageRoleAssistant,
		Content:        answer,
		Sources:        sourcesJSON,
	}
	if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, nil, err
	}

	if conv.Title == "Chat Baru" || conv.Title == "" {
		conv.Title = generateTitle(message)
	}
	if err := uc.convRepo.Update(ctx, conv); err != nil {
		return nil, nil, err
	}

	return userMsg, assistantMsg, nil
}

func (uc *ChatUsecase) ListConversations(
	ctx context.Context,
	userID string,
	page, limit int,
) ([]entity.Conversation, int, error) {
	return uc.convRepo.List(ctx, userID, page, limit)
}

func (uc *ChatUsecase) GetConversation(
	ctx context.Context,
	conversationID, userID string,
) (*entity.Conversation, []entity.Message, error) {
	conv, err := uc.convRepo.FindByIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, err
	}
	if conv == nil {
		return nil, nil, fmt.Errorf("conversation not found")
	}

	messages, err := uc.msgRepo.ListByConversation(ctx, conversationID, 100)
	if err != nil {
		return nil, nil, err
	}

	return conv, messages, nil
}

func (uc *ChatUsecase) DeleteConversation(
	ctx context.Context,
	conversationID, userID string,
) error {
	conv, err := uc.convRepo.FindByIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if conv == nil {
		return fmt.Errorf("conversation not found")
	}

	return uc.convRepo.Delete(ctx, conversationID)
}

func toChatHistory(messages []entity.Message) []document.ChatMessage {
	history := make([]document.ChatMessage, 0, len(messages))
	for _, msg := range messages {
		history = append(history, document.ChatMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}
	return history
}

func buildSourcesJSON(ctx context.Context, chunks []entity.SimilarChunk, docUC DocumentQuerier) []byte {
	if len(chunks) == 0 {
		return nil
	}

	sources := make([]map[string]any, 0, len(chunks))
	for _, chunk := range chunks {
		source := map[string]any{
			"documentId": chunk.DocumentID,
			"chunkIndex": chunk.ChunkIndex,
			"similarity": chunk.Similarity,
			"content":    chunk.Content,
		}
		if name := docUC.GetDocumentOriginalName(ctx, chunk.DocumentID); name != "" {
			source["documentName"] = name
		}
		if len(chunk.Metadata) > 0 {
			var meta entity.ChunkMetadata
			if json.Unmarshal(chunk.Metadata, &meta) == nil && meta.PageNumber > 0 {
				source["pageNumber"] = meta.PageNumber
			}
		}
		sources = append(sources, source)
	}

	data, _ := json.Marshal(sources)
	return data
}

func generateTitle(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "Chat Baru"
	}
	if len(message) > 50 {
		return message[:50] + "..."
	}
	return message
}
