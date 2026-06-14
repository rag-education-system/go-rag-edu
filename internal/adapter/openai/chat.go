package openai

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type ChatClient struct {
	client *openai.Client
	model  string
}

type HistoryMessage struct {
	Role    string
	Content string
}

func NewChatClient(apiKey, model string) *ChatClient {
	return &ChatClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (c *ChatClient) GenerateAnswer(
	ctx context.Context,
	query string,
	docContext string,
	history []HistoryMessage,
) (string, error) {
	systemPrompt := `Anda adalah asisten AI yang membantu menjawab pertanyaan berdasarkan dokumen yang diberikan.

Instruksi:
1. Jawab pertanyaan HANYA berdasarkan konteks dokumen yang diberikan
2. Gunakan riwayat percakapan untuk memahami pertanyaan lanjutan (misalnya "jelaskan poin ketiga" atau "apa maksudnya")
3. Jika informasi tidak ada dalam konteks dokumen, katakan "Maaf, saya tidak menemukan informasi tersebut dalam dokumen"
4. Berikan jawaban yang jelas, ringkas, dan terstruktur
5. Gunakan bahasa Indonesia yang baik dan benar`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	for _, item := range history {
		role := openai.ChatMessageRoleUser
		if item.Role == "assistant" {
			role = openai.ChatMessageRoleAssistant
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: item.Content,
		})
	}

	userPrompt := fmt.Sprintf(`Konteks dari dokumen:
%s

Pertanyaan terbaru: %s`, docContext, query)

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userPrompt,
	})

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.7,
		MaxTokens:   700,
	})

	if err != nil {
		return "", fmt.Errorf("Failed to generate answer: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAi")
	}

	return resp.Choices[0].Message.Content, nil
}
