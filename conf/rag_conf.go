package conf

import (
	"flag"
	"os"
	"strconv"
	"strings"

	"github.com/LittleSongxx/langchaingo/embeddings"
	"github.com/LittleSongxx/langchaingo/vectorstores"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"
)

type RagConf struct {
	EmbeddingType             string `json:"embedding_type"`
	EmbeddingBaseURL          string `json:"embedding_base_url"`
	EmbeddingModelID          string `json:"embedding_model_id"`
	EmbeddingQueryInstruction string `json:"embedding_query_instruction"`
	KnowledgePath             string `json:"knowledge_path"`
	VectorDBType              string `json:"vector_db_type"`

	ChromaURL      string `json:"chroma_url"`
	MilvusURL      string `json:"milvus_url"`
	WeaviateURL    string `json:"weaviate_url"`
	WeaviateScheme string `json:"weaviate_scheme"`

	Space string `json:"space"`

	EmbeddingDimensions int `json:"embedding_dimensions"`
	ChunkSize           int `json:"chunk_size"`
	ChunkOverlap        int `json:"chunk_overlap"`

	PostgresDSN   string `json:"postgres_dsn"`
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`

	MinIOEndpoint  string `json:"minio_endpoint"`
	MinIOAccessKey string `json:"minio_access_key"`
	MinIOSecretKey string `json:"minio_secret_key"`
	MinIOBucket    string `json:"minio_bucket"`
	MinIOUseSSL    bool   `json:"minio_use_ssl"`

	DefaultKnowledgeBase string `json:"default_knowledge_base"`
	DefaultCollection    string `json:"default_collection"`
	KnowledgeAutoMigrate bool   `json:"knowledge_auto_migrate"`
	RerankerBaseURL      string `json:"reranker_base_url"`

	Store          vectorstores.VectorStore `json:"-"`
	Embedder       embeddings.Embedder      `json:"-"`
	MilvusClient   client.Client            `json:"-"`
	WeaviateClient *weaviate.Client         `json:"-"`
}

var (
	RagConfInfo = new(RagConf)

	DefaultSpliter = []string{"\n\n", "\n", " ", ""}
)

func InitRagConf() {
	flag.StringVar(&RagConfInfo.EmbeddingType, "embedding_type", "", "embedding split api: openai gemini ernie huggingface")
	flag.StringVar(&RagConfInfo.EmbeddingBaseURL, "embedding_base_url", "http://localhost:8080", "huggingface text embeddings inference base url")
	flag.StringVar(&RagConfInfo.EmbeddingModelID, "embedding_model_id", "", "huggingface embedding model id")
	flag.StringVar(&RagConfInfo.EmbeddingQueryInstruction, "embedding_query_instruction", "", "query instruction prepended before query embedding")
	flag.StringVar(&RagConfInfo.KnowledgePath, "knowledge_path", GetAbsPath("data/knowledge"), "knowledge")
	flag.StringVar(&RagConfInfo.VectorDBType, "vector_db_type", "milvus", "vector db type: chroma weaviate milvus")

	flag.StringVar(&RagConfInfo.ChromaURL, "chroma_url", "http://localhost:8000", "chroma url")
	flag.StringVar(&RagConfInfo.MilvusURL, "milvus_url", "localhost:19530", "milvus url")
	flag.StringVar(&RagConfInfo.WeaviateURL, "weaviate_url", "localhost:8000", "weaviate url localhost:8000")
	flag.StringVar(&RagConfInfo.WeaviateScheme, "weaviate_scheme", "http", "weaviate scheme: http")
	flag.StringVar(&RagConfInfo.Space, "space", "TinyClaw", "chroma space")

	flag.IntVar(&RagConfInfo.EmbeddingDimensions, "embedding_dimensions", 512, "embedding dimensions for pgvector")
	flag.IntVar(&RagConfInfo.ChunkSize, "chunk_size", 500, "rag file chunk size")
	flag.IntVar(&RagConfInfo.ChunkOverlap, "chunk_overlap", 50, "rag file chunk overlap")

	flag.StringVar(&RagConfInfo.PostgresDSN, "postgres_dsn", "", "postgres dsn for agent runtime and rag v2")
	flag.StringVar(&RagConfInfo.RedisAddr, "redis_addr", "", "redis addr for rag ingestion queue")
	flag.StringVar(&RagConfInfo.RedisPassword, "redis_password", "", "redis password for rag ingestion queue")
	flag.IntVar(&RagConfInfo.RedisDB, "redis_db", 0, "redis db for rag ingestion queue")

	flag.StringVar(&RagConfInfo.MinIOEndpoint, "minio_endpoint", "", "minio endpoint for rag v2 object storage")
	flag.StringVar(&RagConfInfo.MinIOAccessKey, "minio_access_key", "", "minio access key")
	flag.StringVar(&RagConfInfo.MinIOSecretKey, "minio_secret_key", "", "minio secret key")
	flag.StringVar(&RagConfInfo.MinIOBucket, "minio_bucket", "tinyclaw-knowledge", "minio bucket for rag v2")
	flag.BoolVar(&RagConfInfo.MinIOUseSSL, "minio_use_ssl", false, "use ssl to access minio")

	flag.StringVar(&RagConfInfo.DefaultKnowledgeBase, "default_knowledge_base", "default", "default knowledge base name")
	flag.StringVar(&RagConfInfo.DefaultCollection, "default_collection", "default", "default knowledge collection name")
	flag.BoolVar(&RagConfInfo.KnowledgeAutoMigrate, "knowledge_auto_migrate", true, "auto migrate legacy rag files to rag v2")
	flag.StringVar(&RagConfInfo.RerankerBaseURL, "reranker_base_url", "", "optional reranker base url")

}

func EnvRagConf() {
	if os.Getenv("EMBEDDING_TYPE") != "" {
		RagConfInfo.EmbeddingType = os.Getenv("EMBEDDING_TYPE")
	}

	if os.Getenv("EMBEDDING_BASE_URL") != "" {
		RagConfInfo.EmbeddingBaseURL = os.Getenv("EMBEDDING_BASE_URL")
	}

	if os.Getenv("EMBEDDING_MODEL_ID") != "" {
		RagConfInfo.EmbeddingModelID = os.Getenv("EMBEDDING_MODEL_ID")
	}

	if os.Getenv("EMBEDDING_QUERY_INSTRUCTION") != "" {
		RagConfInfo.EmbeddingQueryInstruction = os.Getenv("EMBEDDING_QUERY_INSTRUCTION")
	}

	if os.Getenv("KNOWLEDGE_PATH") != "" {
		RagConfInfo.KnowledgePath = os.Getenv("KNOWLEDGE_PATH")
	}

	if os.Getenv("VECTOR_DB_TYPE") != "" {
		RagConfInfo.VectorDBType = os.Getenv("VECTOR_DB_TYPE")
	}

	if os.Getenv("CHROMA_URL") != "" {
		RagConfInfo.ChromaURL = os.Getenv("CHROMA_URL")
	}

	if os.Getenv("MILVUS_URL") != "" {
		RagConfInfo.MilvusURL = os.Getenv("MILVUS_URL")
	}

	if os.Getenv("WEAVIATE_SCHEME") != "" {
		RagConfInfo.WeaviateScheme = os.Getenv("WEAVIATE_SCHEME")
	}

	if os.Getenv("WEAVIATE_URL") != "" {
		RagConfInfo.WeaviateURL = os.Getenv("WEAVIATE_URL")
	}

	if os.Getenv("SPACE") != "" {
		RagConfInfo.Space = os.Getenv("SPACE")
	}

	if os.Getenv("EMBEDDING_DIMENSIONS") != "" {
		RagConfInfo.EmbeddingDimensions, _ = strconv.Atoi(os.Getenv("EMBEDDING_DIMENSIONS"))
	}

	if os.Getenv("CHUNK_SIZE") != "" {
		RagConfInfo.ChunkSize, _ = strconv.Atoi(os.Getenv("CHUNK_SIZE"))
	}

	if os.Getenv("CHUNK_OVERLAP") != "" {
		RagConfInfo.ChunkOverlap, _ = strconv.Atoi(os.Getenv("CHUNK_OVERLAP"))
	}

	if os.Getenv("POSTGRES_DSN") != "" {
		RagConfInfo.PostgresDSN = os.Getenv("POSTGRES_DSN")
	}

	if os.Getenv("REDIS_ADDR") != "" {
		RagConfInfo.RedisAddr = os.Getenv("REDIS_ADDR")
	}

	if os.Getenv("REDIS_PASSWORD") != "" {
		RagConfInfo.RedisPassword = os.Getenv("REDIS_PASSWORD")
	}

	if os.Getenv("REDIS_DB") != "" {
		RagConfInfo.RedisDB, _ = strconv.Atoi(os.Getenv("REDIS_DB"))
	}

	if os.Getenv("MINIO_ENDPOINT") != "" {
		RagConfInfo.MinIOEndpoint = os.Getenv("MINIO_ENDPOINT")
	}

	if os.Getenv("MINIO_ACCESS_KEY") != "" {
		RagConfInfo.MinIOAccessKey = os.Getenv("MINIO_ACCESS_KEY")
	}

	if os.Getenv("MINIO_SECRET_KEY") != "" {
		RagConfInfo.MinIOSecretKey = os.Getenv("MINIO_SECRET_KEY")
	}

	if os.Getenv("MINIO_BUCKET") != "" {
		RagConfInfo.MinIOBucket = os.Getenv("MINIO_BUCKET")
	}

	if os.Getenv("MINIO_USE_SSL") != "" {
		RagConfInfo.MinIOUseSSL = os.Getenv("MINIO_USE_SSL") == "true"
	}

	if os.Getenv("DEFAULT_KNOWLEDGE_BASE") != "" {
		RagConfInfo.DefaultKnowledgeBase = os.Getenv("DEFAULT_KNOWLEDGE_BASE")
	}

	if os.Getenv("DEFAULT_COLLECTION") != "" {
		RagConfInfo.DefaultCollection = os.Getenv("DEFAULT_COLLECTION")
	}

	if os.Getenv("KNOWLEDGE_AUTO_MIGRATE") != "" {
		RagConfInfo.KnowledgeAutoMigrate = os.Getenv("KNOWLEDGE_AUTO_MIGRATE") == "true"
	}

	if os.Getenv("RERANKER_BASE_URL") != "" {
		RagConfInfo.RerankerBaseURL = os.Getenv("RERANKER_BASE_URL")
	}
}

func (r *RagConf) UseKnowledgeV2() bool {
	if r == nil {
		return false
	}

	return strings.TrimSpace(r.PostgresDSN) != "" &&
		strings.TrimSpace(r.RedisAddr) != "" &&
		strings.TrimSpace(r.MinIOEndpoint) != ""
}

func (r *RagConf) KnowledgeBaseName() string {
	if r == nil || strings.TrimSpace(r.DefaultKnowledgeBase) == "" {
		return "default"
	}
	return strings.TrimSpace(r.DefaultKnowledgeBase)
}

func (r *RagConf) CollectionName() string {
	if r == nil || strings.TrimSpace(r.DefaultCollection) == "" {
		return "default"
	}
	return strings.TrimSpace(r.DefaultCollection)
}
