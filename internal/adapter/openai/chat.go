package openai

import (
	"context"
	"fmt"
	"io"

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
	systemPrompt := `Anda adalah asisten AI edukatif yang membantu menjawab pertanyaan. Anda memiliki akses ke dokumen pembelajaran pengguna DAN pengetahuan umum Anda.

## Prioritas Jawaban:
1. **UTAMAKAN** informasi dari konteks dokumen yang diberikan
2. **LENGKAPI** dengan pengetahuan umum Anda jika dokumen tidak lengkap atau kurang detail
3. **BEDAKAN** dengan jelas mana yang dari dokumen dan mana dari pengetahuan umum

## Format Jawaban:
- Jika menjawab dari dokumen: "Berdasarkan dokumen [nama file], ..." — gunakan nama file persis seperti label [Dokumen: ...] di konteks, bukan nomor urut
- Jika melengkapi dengan pengetahuan umum: "Sebagai tambahan informasi umum, ..." atau "Untuk melengkapi, secara umum..."
- Jika dokumen tidak relevan sama sekali: Jawab dengan pengetahuan Anda, tapi sebutkan bahwa dokumen tidak membahas topik tersebut

## Instruksi Detail:
1. Gunakan riwayat percakapan untuk memahami pertanyaan lanjutan (misalnya "jelaskan poin ketiga", "apa solusinya", atau "masalah tersebut").
2. Jika pengguna menanyakan solusi/upaya/metode setelah sebelumnya membahas masalah dari suatu dokumen, cari di konteks dokumen bagian tujuan, metode, rancangan, implementasi, atau solusi yang terkait topik percakapan sebelumnya.
3. Jika konteks dokumen menyebut topik secara sebagian, jelaskan apa yang ada di dokumen, lalu lengkapi dengan pengetahuan umum jika diperlukan.
4. Jika ada catatan teks rusak atau placeholder "[rumus/simbol tidak terbaca]", jangan mengutip teks rumus yang rusak; cukup jelaskan bahwa contoh rumus tidak terbaca dan berikan penjelasan umum jika memungkinkan.
5. Jika ada petunjuk bahwa pengguna menanyakan aplikasi berbeda (misalnya Google Docs vs Microsoft Word), jelaskan perbedaan itu secara eksplisit.
6. Berikan jawaban yang jelas, ringkas, dan terstruktur dalam bahasa Indonesia yang baik dan benar. Gunakan format **Markdown** (heading, bullet/numbered list, bold, code block) agar mudah dibaca.
7. Untuk pertanyaan tentang langkah-langkah/tutorial yang tidak ada di dokumen, Anda BOLEH memberikan panduan umum dengan catatan bahwa itu bukan dari dokumen.
8. Jika informasi berasal dari beberapa dokumen, sebutkan masing-masing sumbernya untuk transparansi.
9. Untuk pertanyaan faktual/data spesifik (seperti angka, tanggal, nama), prioritaskan data dari dokumen. Jangan mengarang data.`

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
		MaxTokens:   1200,
	})

	if err != nil {
		return "", fmt.Errorf("Failed to generate answer: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAi")
	}

	return resp.Choices[0].Message.Content, nil
}

// GenerateAnswerStream streams the answer token-by-token (SSE pattern from AI-Hukum-BE).
func (c *ChatClient) GenerateAnswerStream(
	ctx context.Context,
	query string,
	docContext string,
	history []HistoryMessage,
) (<-chan string, <-chan error) {
	contentCh := make(chan string, 32)
	errCh := make(chan error, 1)

	go func() {
		defer close(contentCh)
		defer close(errCh)

		systemPrompt := `Anda adalah asisten AI edukatif yang membantu menjawab pertanyaan. Anda memiliki akses ke dokumen pembelajaran pengguna DAN pengetahuan umum Anda.

## Prioritas Jawaban:
1. **UTAMAKAN** informasi dari konteks dokumen yang diberikan
2. **LENGKAPI** dengan pengetahuan umum Anda jika dokumen tidak lengkap atau kurang detail
3. **BEDAKAN** dengan jelas mana yang dari dokumen dan mana dari pengetahuan umum

## Format Jawaban:
- Jika menjawab dari dokumen: "Berdasarkan dokumen [nama file], ..." — gunakan nama file persis seperti label [Dokumen: ...] di konteks
- Jika melengkapi dengan pengetahuan umum: "Sebagai tambahan informasi umum, ..."
- Untuk pertanyaan lanjutan tentang solusi/metode, prioritaskan bagian dokumen tentang tujuan, metode, rancangan, atau implementasi terkait topik sebelumnya
- Berikan jawaban yang jelas, ringkas, dan terstruktur dalam bahasa Indonesia yang baik dan benar. Gunakan format **Markdown** (heading, bullet/numbered list, bold, code block) agar mudah dibaca`

		messages := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
		}
		for _, item := range history {
			role := openai.ChatMessageRoleUser
			if item.Role == "assistant" {
				role = openai.ChatMessageRoleAssistant
			}
			messages = append(messages, openai.ChatCompletionMessage{Role: role, Content: item.Content})
		}

		userPrompt := fmt.Sprintf(`Konteks dari dokumen:
%s

Pertanyaan terbaru: %s`, docContext, query)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userPrompt,
		})

		stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:       c.model,
			Messages:    messages,
			Temperature: 0.2,
			MaxTokens:   1200,
			Stream:      true,
		})
		if err != nil {
			errCh <- fmt.Errorf("failed to start stream: %w", err)
			return
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}
				errCh <- err
				return
			}
			if len(response.Choices) > 0 {
				token := response.Choices[0].Delta.Content
				if token != "" {
					contentCh <- token
				}
			}
		}
	}()

	return contentCh, errCh
}
