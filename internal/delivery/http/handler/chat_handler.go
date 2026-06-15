package handler

import (
	"bufio"
	"encoding/json"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/chat"
	"rag-api/internal/usecase/document"
)

type ChatHandler struct {
	chatUsecase *chat.ChatUsecase
	docUsecase  *document.DocumentUsecase
}

func NewChatHandler(chatUsecase *chat.ChatUsecase, docUsecase *document.DocumentUsecase) *ChatHandler {
	return &ChatHandler{chatUsecase: chatUsecase, docUsecase: docUsecase}
}

func (h *ChatHandler) CreateConversation(c *fiber.Ctx) error {
	access := documentAccessFromCtx(c)

	var req dto.CreateConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	conv, userMsg, assistantMsg, err := h.chatUsecase.CreateConversation(c.Context(), access, req.Message)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.ChatResponse{
		ConversationID: conv.ID,
		UserMessage:  toMessageResponse(userMsg),
		AssistantMessage: toMessageResponse(assistantMsg, h.docUsecase),
	})
}

func (h *ChatHandler) SendMessage(c *fiber.Ctx) error {
	access := documentAccessFromCtx(c)
	conversationID := c.Params("id")

	var req dto.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	userMsg, assistantMsg, err := h.chatUsecase.SendMessage(c.Context(), conversationID, access, req.Message)
	if err != nil {
		if err.Error() == "conversation not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(dto.ChatResponse{
		ConversationID: conversationID,
		UserMessage:    toMessageResponse(userMsg),
		AssistantMessage: toMessageResponse(assistantMsg, h.docUsecase),
	})
}

func (h *ChatHandler) ListConversations(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	convs, total, err := h.chatUsecase.ListConversations(c.Context(), userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	convInfos := make([]dto.ConversationInfo, 0, len(convs))
	for _, conv := range convs {
		convInfos = append(convInfos, dto.ConversationInfo{
			ID:        conv.ID,
			Title:     conv.Title,
			CreatedAt: conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data": convInfos,
		"meta": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *ChatHandler) GetConversation(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	conversationID := c.Params("id")

	conv, messages, err := h.chatUsecase.GetConversation(c.Context(), conversationID, userID)
	if err != nil {
		if err.Error() == "conversation not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	msgResponses := make([]dto.ChatMessageResponse, 0, len(messages))
	for _, msg := range messages {
		msgResponses = append(msgResponses, toMessageResponse(&msg, h.docUsecase))
	}

	return c.Status(fiber.StatusOK).JSON(dto.ConversationDetail{
		Conversation: dto.ConversationInfo{
			ID:        conv.ID,
			Title:     conv.Title,
			CreatedAt: conv.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: conv.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
		Messages: msgResponses,
	})
}

func (h *ChatHandler) DeleteConversation(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	conversationID := c.Params("id")

	if err := h.chatUsecase.DeleteConversation(c.Context(), conversationID, userID); err != nil {
		if err.Error() == "conversation not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Conversation deleted successfully"})
}

// StreamChat streams RAG chat response via SSE (pattern from AI-Hukum-BE).
func (h *ChatHandler) StreamChat(c *fiber.Ctx) error {
	access := documentAccessFromCtx(c)

	var req dto.StreamChatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if req.Message == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "message is required"})
	}

	streamChan, err := h.chatUsecase.ChatStream(
		c.Context(),
		access,
		req.ConversationID,
		req.Message,
		document.ParseChatMode(req.ChatMode),
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		w.Write([]byte(": connected\n\n"))
		w.Flush()

		for chunk := range streamChan {
			data, err := json.Marshal(chunk)
			if err != nil {
				errChunk, _ := json.Marshal(dto.StreamChunk{Type: "error", Error: "marshal failed"})
				w.Write([]byte("data: "))
				w.Write(errChunk)
				w.Write([]byte("\n\n"))
				w.Flush()
				break
			}
			w.Write([]byte("data: "))
			w.Write(data)
			w.Write([]byte("\n\n"))
			w.Flush()

			if chunk.Type == "error" || chunk.Type == "done" {
				break
			}
		}
	})

	return nil
}

func toMessageResponse(msg *entity.Message, docUC ...*document.DocumentUsecase) dto.ChatMessageResponse {
	resp := dto.ChatMessageResponse{
		ID:        msg.ID,
		Role:      string(msg.Role),
		Content:   msg.Content,
		CreatedAt: msg.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if len(msg.Sources) > 0 {
		var rawSources []map[string]any
		if json.Unmarshal(msg.Sources, &rawSources) == nil {
			for _, raw := range rawSources {
				source := dto.ChunkSource{}
				if v, ok := raw["documentId"].(string); ok {
					source.DocumentID = v
				}
				if v, ok := raw["documentName"].(string); ok {
					source.DocumentName = v
				}
				if v, ok := raw["similarity"].(float64); ok {
					source.Similarity = v
				}
				if v, ok := raw["content"].(string); ok {
					source.Content = v
				}
				if v, ok := raw["chunkIndex"].(float64); ok {
					source.ChunkIndex = int(v)
				}
				if v, ok := raw["pageNumber"].(float64); ok {
					source.PageNumber = int(v)
				}
				if len(docUC) > 0 && docUC[0] != nil {
					source.LowConfidence = document.IsLowConfidenceSource(source.Similarity, source.Content)
				}
				resp.Sources = append(resp.Sources, source)
			}
		}
	}

	return resp
}
