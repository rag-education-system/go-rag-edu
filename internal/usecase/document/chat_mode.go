package document

import (
	"regexp"
	"strings"
)

type ChatMode string

const (
	ChatModeHybrid ChatMode = "hybrid"
	ChatModeStrict ChatMode = "strict"
)

type ResponseStrategy string

const (
	ResponseFromDocuments    ResponseStrategy = "document"
	ResponseGeneralKnowledge ResponseStrategy = "general"
	ResponseRefusal          ResponseStrategy = "refusal"
)

type RAGResponsePlan struct {
	UseLLM       bool
	DirectAnswer string
	DocContext   string
	Strategy     ResponseStrategy
}

var factualQueryPattern = regexp.MustCompile(`(?i)\b(berapa|brp|kapan|tanggal|jadwal|jam\s*ke|hari\s*(apa|ini)|sk[sS]|nilai|nomor|nip|nim|nama\s+dosen|siapa\s+dosen|siapa\s+pengampu|biaya|harga|rupiah|tahun\s*ajaran|semester|uts|uas|deadline|batas\s+waktu|tanggal\s+ujian|berapa\s+lama|berapa\s+banyak)\b`)

func ParseChatMode(raw string) ChatMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "strict", "document", "document_only":
		return ChatModeStrict
	default:
		return ChatModeHybrid
	}
}

func HasRelevantDocuments(rag *RAGResult) bool {
	return rag != nil && rag.DocContext != "" && len(rag.Chunks) > 0
}

func IsFactualQuery(query string) bool {
	return factualQueryPattern.MatchString(strings.TrimSpace(query))
}

func StrictRefusalMessage() string {
	return "Maaf, saya tidak menemukan materi yang relevan di dokumen yang tersedia. Silakan unggah slide atau modul terkait, atau ubah pertanyaan Anda agar sesuai dengan dokumen yang ada."
}

func HybridFactualRefusalMessage() string {
	return "Saya tidak menemukan informasi tersebut di dokumen Anda. Untuk pertanyaan faktual (angka, jadwal, nama, atau data spesifik), saya tidak dapat memberikan jawaban tanpa sumber dokumen. Silakan unggah materi terkait atau periksa kembali dokumen Anda."
}

func PlanRAGResponse(mode ChatMode, rag *RAGResult, query string) RAGResponsePlan {
	if rag != nil && rag.Answer != "" {
		return RAGResponsePlan{
			UseLLM:       false,
			DirectAnswer: rag.Answer,
			Strategy:     ResponseFromDocuments,
		}
	}

	if HasRelevantDocuments(rag) {
		docContext := rag.DocContext
		if mode == ChatModeStrict {
			docContext = "[MODE KETAT: Jawab HANYA berdasarkan konteks dokumen di bawah. Jika informasi tidak ada di konteks, katakan dengan jelas bahwa dokumen tidak memuat informasi tersebut. Jangan gunakan pengetahuan umum di luar dokumen.]\n\n" + docContext
		}
		return RAGResponsePlan{
			UseLLM:     true,
			DocContext: docContext,
			Strategy:   ResponseFromDocuments,
		}
	}

	if mode == ChatModeStrict {
		return RAGResponsePlan{
			UseLLM:       false,
			DirectAnswer: StrictRefusalMessage(),
			Strategy:     ResponseRefusal,
		}
	}

	if IsFactualQuery(query) {
		return RAGResponsePlan{
			UseLLM:       false,
			DirectAnswer: HybridFactualRefusalMessage(),
			Strategy:     ResponseRefusal,
		}
	}

	return RAGResponsePlan{
		UseLLM: true,
		DocContext: "[Tidak ada dokumen relevan yang ditemukan di dokumen pengguna. " +
			"Awali jawaban dengan kalimat bahwa materi ini tidak ditemukan di dokumen mereka. " +
			"Lalu jelaskan secara umum dengan pengetahuan Anda, dan tegaskan bahwa jawaban ini bukan bersumber dari dokumen pengguna.]",
		Strategy: ResponseGeneralKnowledge,
	}
}
