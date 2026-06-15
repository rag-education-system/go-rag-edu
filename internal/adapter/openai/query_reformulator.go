package openai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rag-api/internal/usecase/document"

	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultHistoryMessages = 10
	maxHistoryTokens       = 1500
	maxAssistantMsgChars   = 300
)

// QueryReformulator rewrites user queries using conversation context (pattern from AI-Hukum-BE).
type QueryReformulator struct {
	client          *openai.Client
	model           string
	timeout         time.Duration
	enabled         bool
	historyMessages int
}

func NewQueryReformulator(apiKey, model string, timeout time.Duration, enabled bool) *QueryReformulator {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &QueryReformulator{
		client:          openai.NewClient(apiKey),
		model:           model,
		timeout:         timeout,
		enabled:         enabled,
		historyMessages: defaultHistoryMessages,
	}
}

func (r *QueryReformulator) Enabled() bool {
	return r.enabled
}

func (r *QueryReformulator) ReformulateQuery(ctx context.Context, query string, history []document.ChatMessage) (string, error) {
	if !r.enabled || strings.TrimSpace(query) == "" {
		return query, nil
	}

	contextStr := buildConversationContext(history, r.historyMessages, maxHistoryTokens)

	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	systemPrompt, userPrompt := buildEducationReformulationPrompts(contextStr, query)

	resp, err := r.client.CreateChatCompletion(timeoutCtx, openai.ChatCompletionRequest{
		Model:       r.model,
		Temperature: 0.1,
		MaxTokens:   300,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
	})
	if err != nil {
		return query, fmt.Errorf("LLM reformulation failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return query, fmt.Errorf("LLM returned empty response")
	}

	reformulated := strings.TrimSpace(resp.Choices[0].Message.Content)
	if reformulated == "" {
		return query, nil
	}
	return reformulated, nil
}

func buildConversationContext(history []document.ChatMessage, lastN, maxTokens int) string {
	if len(history) == 0 {
		return ""
	}

	start := len(history) - lastN
	if start < 0 {
		start = 0
	}

	var parts []string
	estimatedTokens := 0

	for i := start; i < len(history); i++ {
		msg := history[i]
		role := "Pengguna"
		content := msg.Content
		if msg.Role == "assistant" {
			role = "Asisten"
			if len(content) > maxAssistantMsgChars {
				runes := []rune(content)
				content = string(runes[:maxAssistantMsgChars]) + "..."
			}
		}

		msgTokens := len(content) / 4
		if estimatedTokens+msgTokens > maxTokens {
			break
		}
		parts = append(parts, fmt.Sprintf("%s: %s", role, content))
		estimatedTokens += msgTokens
	}

	return strings.Join(parts, "\n")
}

func buildEducationReformulationPrompts(context, query string) (string, string) {
	if context != "" {
		systemPrompt := `Anda adalah query reformulator untuk sistem pencarian dokumen pembelajaran edukatif.

TUGAS: Tulis ulang pertanyaan pengguna menjadi query pencarian yang mempertahankan konteks percakapan sebelumnya.

ATURAN:
1. Ganti kata ganti (ini, itu, tersebut, poin ketiga, yang tadi) dengan referensi konkret dari konteks
2. Pertahankan kesinambungan topik jika pertanyaan masih terkait
3. Jika query sudah lengkap, kembalikan tanpa perubahan signifikan
4. Jangan menjawab pertanyaan — hanya tulis ulang query
5. Output HANYA query hasil reformulasi, tanpa penjelasan atau tanda kutip

CONTOH:
Konteks: "Pengguna: Apa itu fotosintesis? Asisten: Fotosintesis adalah proses..."
Query: "Jelaskan tahapannya"
Output: Jelaskan tahapan proses fotosintesis pada tumbuhan`

		userPrompt := fmt.Sprintf(`KONTEKS PERCAKAPAN:
%s

QUERY BARU:
%s

Tulis ulang query di atas. Output hanya query hasil reformulasi.`, context, query)
		return systemPrompt, userPrompt
	}

	systemPrompt := `Anda adalah query reformulator untuk sistem pencarian dokumen pembelajaran.

TUGAS: Perbaiki pertanyaan pengguna menjadi query pencarian yang optimal untuk menemukan dokumen edukatif yang relevan.

ATURAN:
1. Perluas query singkat dengan kata kunci edukatif yang relevan
2. Gunakan terminologi yang tepat untuk pencarian dokumen
3. Pertahankan maksud asli pertanyaan
4. Output HANYA query yang sudah diperbaiki, tanpa penjelasan

CONTOH:
Query: "sel fotosintesis"
Output: Apa fungsi dan struktur sel pada proses fotosintesis pada tumbuhan?`

	userPrompt := fmt.Sprintf(`QUERY PENGGUNA:
%s

Perbaiki query di atas untuk pencarian dokumen edukatif. Output hanya query hasil perbaikan.`, query)
	return systemPrompt, userPrompt
}
