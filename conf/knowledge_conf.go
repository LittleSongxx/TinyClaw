package conf

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/LittleSongxx/langchaingo/embeddings"
)

type KnowledgeConf struct {
	EmbeddingType             string `json:"embedding_type"`
	EmbeddingBaseURL          string `json:"embedding_base_url"`
	EmbeddingModelID          string `json:"embedding_model_id"`
	EmbeddingQueryInstruction string `json:"embedding_query_instruction"`
	KnowledgePath             string `json:"knowledge_path"`
	RerankerBaseURL           string `json:"reranker_base_url"`

	EmbeddingDimensions int `json:"embedding_dimensions"`
	ChunkSize           int `json:"chunk_size"`
	ChunkOverlap        int `json:"chunk_overlap"`
	RerankerTopN        int `json:"reranker_top_n"`

	PostgresDSN   string `json:"postgres_dsn"`
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	MinIOEndpoint  string `json:"minio_endpoint"`
	MinIOAccessKey string `json:"minio_access_key"`
	MinIOSecretKey string `json:"minio_secret_key"`
	MinIOBucket    string `json:"minio_bucket"`
	MinIOUseSSL    bool   `json:"minio_use_ssl"`

	DefaultKnowledgeBase   string  `json:"default_knowledge_base"`
	DefaultCollection      string  `json:"default_collection"`
	KnowledgeAutoMigrate   bool    `json:"knowledge_auto_migrate"`
	DenseScoreThreshold    float64 `json:"dense_score_threshold"`
	LexicalScoreThreshold  float64 `json:"lexical_score_threshold"`
	FusedScoreThreshold    float64 `json:"fused_score_threshold"`
	RerankerScoreThreshold float64 `json:"reranker_score_threshold"`

	Embedder embeddings.Embedder `json:"-"`
}

var (
	KnowledgeConfInfo = new(KnowledgeConf)

	DefaultSpliter = []string{"\n\n", "\n", " ", ""}
)

func InitKnowledgeConf() {
	flag.StringVar(&KnowledgeConfInfo.EmbeddingType, "embedding_type", "", "embedding split api: openai gemini ernie huggingface")
	flag.StringVar(&KnowledgeConfInfo.EmbeddingBaseURL, "embedding_base_url", "http://localhost:8080", "huggingface text embeddings inference base url")
	flag.StringVar(&KnowledgeConfInfo.EmbeddingModelID, "embedding_model_id", "", "huggingface embedding model id")
	flag.StringVar(&KnowledgeConfInfo.EmbeddingQueryInstruction, "embedding_query_instruction", "", "query instruction prepended before query embedding")
	flag.StringVar(&KnowledgeConfInfo.KnowledgePath, "knowledge_path", GetAbsPath("data/knowledge"), "knowledge")

	flag.IntVar(&KnowledgeConfInfo.EmbeddingDimensions, "embedding_dimensions", 512, "embedding dimensions for pgvector")
	flag.IntVar(&KnowledgeConfInfo.ChunkSize, "chunk_size", 500, "knowledge document chunk size")
	flag.IntVar(&KnowledgeConfInfo.ChunkOverlap, "chunk_overlap", 50, "knowledge document chunk overlap")
	flag.IntVar(&KnowledgeConfInfo.RerankerTopN, "reranker_top_n", 8, "top N fused hits sent to the reranker")

	flag.StringVar(&KnowledgeConfInfo.PostgresDSN, "postgres_dsn", "", "postgres dsn for knowledge storage")
	flag.StringVar(&KnowledgeConfInfo.RedisAddr, "redis_addr", "", "redis addr for optional knowledge ingestion queue")
	flag.StringVar(&KnowledgeConfInfo.RedisPassword, "redis_password", "", "redis password for optional knowledge ingestion queue")
	flag.IntVar(&KnowledgeConfInfo.RedisDB, "redis_db", 0, "redis db for optional knowledge ingestion queue")

	flag.StringVar(&KnowledgeConfInfo.MinIOEndpoint, "minio_endpoint", "", "minio endpoint for knowledge object storage")
	flag.StringVar(&KnowledgeConfInfo.MinIOAccessKey, "minio_access_key", "", "minio access key")
	flag.StringVar(&KnowledgeConfInfo.MinIOSecretKey, "minio_secret_key", "", "minio secret key")
	flag.StringVar(&KnowledgeConfInfo.MinIOBucket, "minio_bucket", "tinyclaw-knowledge", "minio bucket for knowledge storage")
	flag.BoolVar(&KnowledgeConfInfo.MinIOUseSSL, "minio_use_ssl", false, "use ssl to access minio")

	flag.StringVar(&KnowledgeConfInfo.DefaultKnowledgeBase, "default_knowledge_base", "default", "default knowledge base name")
	flag.StringVar(&KnowledgeConfInfo.DefaultCollection, "default_collection", "default", "default knowledge collection name")
	flag.BoolVar(&KnowledgeConfInfo.KnowledgeAutoMigrate, "knowledge_auto_migrate", true, "auto migrate legacy knowledge files into the unified knowledge store")
	flag.StringVar(&KnowledgeConfInfo.RerankerBaseURL, "reranker_base_url", "", "optional reranker base url")
	flag.Float64Var(&KnowledgeConfInfo.DenseScoreThreshold, "dense_score_threshold", 0.55, "minimum dense similarity score before a retrieved chunk is considered relevant")
	flag.Float64Var(&KnowledgeConfInfo.LexicalScoreThreshold, "lexical_score_threshold", 0.05, "minimum lexical score before a retrieved chunk is considered relevant")
	flag.Float64Var(&KnowledgeConfInfo.FusedScoreThreshold, "fused_score_threshold", 0.02, "minimum fused retrieval score before a retrieved chunk is considered relevant")
	flag.Float64Var(&KnowledgeConfInfo.RerankerScoreThreshold, "reranker_score_threshold", 0.15, "minimum reranker score before a retrieved chunk is considered relevant")

}

func EnvKnowledgeConf() {
	if os.Getenv("EMBEDDING_TYPE") != "" {
		KnowledgeConfInfo.EmbeddingType = os.Getenv("EMBEDDING_TYPE")
	}

	if os.Getenv("EMBEDDING_BASE_URL") != "" {
		KnowledgeConfInfo.EmbeddingBaseURL = os.Getenv("EMBEDDING_BASE_URL")
	}

	if os.Getenv("EMBEDDING_MODEL_ID") != "" {
		KnowledgeConfInfo.EmbeddingModelID = os.Getenv("EMBEDDING_MODEL_ID")
	}

	if os.Getenv("EMBEDDING_QUERY_INSTRUCTION") != "" {
		KnowledgeConfInfo.EmbeddingQueryInstruction = os.Getenv("EMBEDDING_QUERY_INSTRUCTION")
	}

	if os.Getenv("KNOWLEDGE_PATH") != "" {
		KnowledgeConfInfo.KnowledgePath = os.Getenv("KNOWLEDGE_PATH")
	}

	if os.Getenv("EMBEDDING_DIMENSIONS") != "" {
		KnowledgeConfInfo.EmbeddingDimensions, _ = strconv.Atoi(os.Getenv("EMBEDDING_DIMENSIONS"))
	}

	if os.Getenv("CHUNK_SIZE") != "" {
		KnowledgeConfInfo.ChunkSize, _ = strconv.Atoi(os.Getenv("CHUNK_SIZE"))
	}

	if os.Getenv("CHUNK_OVERLAP") != "" {
		KnowledgeConfInfo.ChunkOverlap, _ = strconv.Atoi(os.Getenv("CHUNK_OVERLAP"))
	}

	if os.Getenv("RERANKER_TOP_N") != "" {
		KnowledgeConfInfo.RerankerTopN, _ = strconv.Atoi(os.Getenv("RERANKER_TOP_N"))
	}

	if os.Getenv("POSTGRES_DSN") != "" {
		KnowledgeConfInfo.PostgresDSN = os.Getenv("POSTGRES_DSN")
	}

	if os.Getenv("REDIS_ADDR") != "" {
		KnowledgeConfInfo.RedisAddr = os.Getenv("REDIS_ADDR")
	}

	if os.Getenv("REDIS_PASSWORD") != "" {
		KnowledgeConfInfo.RedisPassword = os.Getenv("REDIS_PASSWORD")
	}

	if os.Getenv("REDIS_DB") != "" {
		KnowledgeConfInfo.RedisDB, _ = strconv.Atoi(os.Getenv("REDIS_DB"))
	}

	if os.Getenv("MINIO_ENDPOINT") != "" {
		KnowledgeConfInfo.MinIOEndpoint = os.Getenv("MINIO_ENDPOINT")
	}

	if os.Getenv("MINIO_ACCESS_KEY") != "" {
		KnowledgeConfInfo.MinIOAccessKey = os.Getenv("MINIO_ACCESS_KEY")
	}

	if os.Getenv("MINIO_SECRET_KEY") != "" {
		KnowledgeConfInfo.MinIOSecretKey = os.Getenv("MINIO_SECRET_KEY")
	}

	if os.Getenv("MINIO_BUCKET") != "" {
		KnowledgeConfInfo.MinIOBucket = os.Getenv("MINIO_BUCKET")
	}

	if os.Getenv("MINIO_USE_SSL") != "" {
		KnowledgeConfInfo.MinIOUseSSL = os.Getenv("MINIO_USE_SSL") == "true"
	}

	if os.Getenv("DEFAULT_KNOWLEDGE_BASE") != "" {
		KnowledgeConfInfo.DefaultKnowledgeBase = os.Getenv("DEFAULT_KNOWLEDGE_BASE")
	}

	if os.Getenv("DEFAULT_COLLECTION") != "" {
		KnowledgeConfInfo.DefaultCollection = os.Getenv("DEFAULT_COLLECTION")
	}

	if os.Getenv("KNOWLEDGE_AUTO_MIGRATE") != "" {
		KnowledgeConfInfo.KnowledgeAutoMigrate = os.Getenv("KNOWLEDGE_AUTO_MIGRATE") == "true"
	}

	if os.Getenv("RERANKER_BASE_URL") != "" {
		KnowledgeConfInfo.RerankerBaseURL = os.Getenv("RERANKER_BASE_URL")
	}

	if os.Getenv("DENSE_SCORE_THRESHOLD") != "" {
		KnowledgeConfInfo.DenseScoreThreshold, _ = strconv.ParseFloat(os.Getenv("DENSE_SCORE_THRESHOLD"), 64)
	}

	if os.Getenv("LEXICAL_SCORE_THRESHOLD") != "" {
		KnowledgeConfInfo.LexicalScoreThreshold, _ = strconv.ParseFloat(os.Getenv("LEXICAL_SCORE_THRESHOLD"), 64)
	}

	if os.Getenv("FUSED_SCORE_THRESHOLD") != "" {
		KnowledgeConfInfo.FusedScoreThreshold, _ = strconv.ParseFloat(os.Getenv("FUSED_SCORE_THRESHOLD"), 64)
	}

	if os.Getenv("RERANKER_SCORE_THRESHOLD") != "" {
		KnowledgeConfInfo.RerankerScoreThreshold, _ = strconv.ParseFloat(os.Getenv("RERANKER_SCORE_THRESHOLD"), 64)
	}
}

func (r *KnowledgeConf) FeatureStoreEnabled() bool {
	if r == nil {
		return false
	}

	return strings.TrimSpace(r.PostgresDSN) != ""
}

func (r *KnowledgeConf) ObjectStorageEnabled() bool {
	if r == nil {
		return false
	}

	return strings.TrimSpace(r.MinIOEndpoint) != ""
}

func (r *KnowledgeConf) QueueEnabled() bool {
	if r == nil {
		return false
	}

	return strings.TrimSpace(r.RedisAddr) != ""
}

func (r *KnowledgeConf) Enabled() bool {
	if r == nil {
		return false
	}

	return r.FeatureStoreEnabled() && r.ObjectStorageEnabled()
}

func (r *KnowledgeConf) KnowledgeBaseName() string {
	if r == nil || strings.TrimSpace(r.DefaultKnowledgeBase) == "" {
		return "default"
	}
	return strings.TrimSpace(r.DefaultKnowledgeBase)
}

func (r *KnowledgeConf) CollectionName() string {
	if r == nil || strings.TrimSpace(r.DefaultCollection) == "" {
		return "default"
	}
	return strings.TrimSpace(r.DefaultCollection)
}
