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
1. Jawab HANYA berdasarkan konteks dokumen. JANGAN menambah langkah menu, fitur, atau pengetahuan umum di luar konteks.
2. DILARANG menyebut tab/menu/langkah seperti "Insert", "Sisipkan", "Review", "Equation Editor", atau langkah numbered tutorial kecuali kata/frasa persis itu muncul di konteks dokumen.
3. Gunakan riwayat percakapan untuk memahami pertanyaan lanjutan (misalnya "jelaskan poin ketiga" atau "apa maksudnya").
4. Jika user minta "cara" tetapi dokumen hanya berisi instruksi latihan (mis. "Buatlah rumus menggunakan fitur Equation"), jelaskan isi dokumen tersebut dan katakan bahwa langkah detail cara membuka/menggunakan fitur tidak tercantum.
5. Jika konteks menyebut topik secara sebagian, jelaskan apa yang disebutkan di dokumen dan bagian mana yang tidak tercantum. JANGAN menolak total jika masih ada informasi sebagian.
6. Jika ada catatan teks rusak atau placeholder "[rumus/simbol tidak terbaca]", jangan mengutip teks rumus yang rusak; cukup jelaskan bahwa contoh rumus tidak terbaca.
7. Jika ada petunjuk bahwa pengguna menanyakan aplikasi berbeda (misalnya Google Docs vs Microsoft Word), jelaskan perbedaan itu secara eksplisit sebelum menjawab.
8. JANGAN katakan "tidak menemukan informasi" jika ada data yang sebagian relevan. Jawab dengan data yang ada, lalu jelaskan jika ada bagian yang tidak tercantum.
9. Berikan jawaban yang jelas, ringkas, dan terstruktur dalam bahasa Indonesia yang baik dan benar.`

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
		Temperature: 0.2,
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
