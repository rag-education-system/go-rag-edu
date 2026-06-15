package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration time.Duration
	Port          int

	// open ai
	OpenAIKey            string
	OpenAIEmbeddingModel string
	OpenAIChatModel      string

	// rag config
	ChunkSize           int
	ChunkOverlap        int
	TopKResults         int
	SimilarityThreshold float64
	UseHybridSearch     bool

	// query reformulation (AI-Hukum-BE pattern)
	QueryReformulationEnabled  bool
	QueryReformulationModel    string
	QueryReformulationTimeout  time.Duration

	// embedding: OpenAI or Ollama local
	IsEmbeddingLocal       bool
	OllamaBaseURL          string
	OllamaEmbeddingModel   string
	OllamaEmbeddingDimension int

	// security / anti-abuse
	BodyLimitMB            int
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	IdleTimeout            time.Duration
	RequestTimeout         time.Duration
	TrustedProxies         []string
	CORSOrigins            []string
	RateLimitGlobalMax     int
	RateLimitGlobalWindow  time.Duration
	RateLimitAuthMax       int
	RateLimitAuthWindow    time.Duration
	RateLimitQueryMax      int
	RateLimitQueryWindow   time.Duration
	RateLimitUploadMax     int
	RateLimitUploadWindow  time.Duration
	RateLimitAdminMax      int
	RateLimitAdminWindow   time.Duration

	// OCR (Tesseract)
	OCREnabled       bool
	OCRLang          string
	OCRMinTextLength int

	// Supabase Storage
	SupabaseURL            string
	SupabaseServiceKey     string
	SupabaseStorageBucket  string
}

func Load() *Config {
	godotenv.Load()
	jwtExp, _ := time.ParseDuration(getEnv("JWT_EXPIRATION", "168h"))

	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		port = 8080
	}

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiration: jwtExp,
		Port:          port,

		OpenAIKey:            getEnv("OPENAI_API_KEY", ""),
		OpenAIEmbeddingModel: getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		OpenAIChatModel:      getEnv("OPENAI_CHAT_MODEL", "gpt-4o-mini"),

		ChunkSize:           getEnvInt("CHUNK_SIZE", 1000),
		ChunkOverlap:        getEnvInt("CHUNK_OVERLAP", 200),
		TopKResults:         getEnvInt("TOP_K_RESULTS", 6),
		SimilarityThreshold: getEnvFloat("SIMILARITY_THRESHOLD", 0.5),
		UseHybridSearch:     getEnvBool("USE_HYBRID_SEARCH", true),

		QueryReformulationEnabled: getEnvBool("QUERY_REFORMULATION_ENABLED", true),
		QueryReformulationModel:   getEnv("QUERY_REFORMULATION_MODEL", "gpt-4o-mini"),
		QueryReformulationTimeout: getEnvDuration("QUERY_REFORMULATION_TIMEOUT", 10*time.Second),

		IsEmbeddingLocal:         getEnvBool("IS_EMBEDDING_LOCAL", false),
		OllamaBaseURL:            getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
		OllamaEmbeddingModel:     getEnv("OLLAMA_EMBEDDING_MODEL", "nomic-embed-text"),
		OllamaEmbeddingDimension: getEnvInt("OLLAMA_EMBEDDING_DIMENSION", 1536),

		BodyLimitMB:           getEnvInt("BODY_LIMIT_MB", 12),
		ReadTimeout:           getEnvDuration("READ_TIMEOUT", 10*time.Second),
		WriteTimeout:          getEnvDuration("WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:           getEnvDuration("IDLE_TIMEOUT", 120*time.Second),
		RequestTimeout:        getEnvDuration("REQUEST_TIMEOUT", 60*time.Second),
		TrustedProxies:        getEnvCSV("TRUSTED_PROXIES"),
		CORSOrigins:           getEnvCSV("CORS_ORIGINS"),
		RateLimitGlobalMax:    getEnvInt("RATE_LIMIT_GLOBAL_MAX", 120),
		RateLimitGlobalWindow: getEnvDuration("RATE_LIMIT_GLOBAL_WINDOW", time.Minute),
		RateLimitAuthMax:      getEnvInt("RATE_LIMIT_AUTH_MAX", 10),
		RateLimitAuthWindow:   getEnvDuration("RATE_LIMIT_AUTH_WINDOW", 15*time.Minute),
		RateLimitQueryMax:     getEnvInt("RATE_LIMIT_QUERY_MAX", 20),
		RateLimitQueryWindow:  getEnvDuration("RATE_LIMIT_QUERY_WINDOW", time.Minute),
		RateLimitUploadMax:    getEnvInt("RATE_LIMIT_UPLOAD_MAX", 5),
		RateLimitUploadWindow: getEnvDuration("RATE_LIMIT_UPLOAD_WINDOW", time.Minute),
		RateLimitAdminMax:     getEnvInt("RATE_LIMIT_ADMIN_MAX", 30),
		RateLimitAdminWindow:  getEnvDuration("RATE_LIMIT_ADMIN_WINDOW", time.Minute),

		OCREnabled:       getEnvBool("OCR_ENABLED", true),
		OCRLang:          getEnv("OCR_LANG", "ind+eng"),
		OCRMinTextLength: getEnvInt("OCR_MIN_TEXT_LENGTH", 80),

		SupabaseURL:           getEnv("SUPABASE_URL", ""),
		SupabaseServiceKey:    getEnv("SUPABASE_SERVICE_KEY", ""),
		SupabaseStorageBucket: getEnv("SUPABASE_STORAGE_BUCKET", "documents"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		switch strings.ToLower(strings.TrimSpace(val)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvCSV(key string) []string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return nil
	}

	parts := strings.Split(val, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
