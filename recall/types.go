package recall

type Corpus string

const (
	CorpusMemory    Corpus = "memory"
	CorpusKnowledge Corpus = "knowledge"
)

type RecallQuery struct {
	Corpus  Corpus `json:"corpus"`
	Query   string `json:"query"`
	UserID  string `json:"user_id,omitempty"`
	SkillID string `json:"skill_id,omitempty"`
	Limit   int    `json:"limit,omitempty"`
}

type RecallHit struct {
	ID       string                 `json:"id"`
	Corpus   Corpus                 `json:"corpus"`
	Source   string                 `json:"source"`
	Title    string                 `json:"title,omitempty"`
	Content  string                 `json:"content"`
	Score    float64                `json:"score,omitempty"`
	Citation string                 `json:"citation,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type MemoryStatus struct {
	Enabled       bool     `json:"enabled"`
	Provider      string   `json:"provider"`
	Available     bool     `json:"available"`
	Mode          string   `json:"mode"`
	Tools         []string `json:"tools,omitempty"`
	LastError     string   `json:"last_error,omitempty"`
	HasGraphStore bool     `json:"has_graph_store"`
}

type KnowledgeStatus struct {
	Enabled                bool    `json:"enabled"`
	Backend                string  `json:"backend"`
	Embedder               string  `json:"embedder,omitempty"`
	VectorStore            string  `json:"vector_store,omitempty"`
	DefaultKnowledgeBase   string  `json:"default_knowledge_base,omitempty"`
	DefaultCollection      string  `json:"default_collection,omitempty"`
	AsyncIngestion         bool    `json:"async_ingestion"`
	ObjectStorage          bool    `json:"object_storage"`
	Queue                  bool    `json:"queue"`
	RerankerEnabled        bool    `json:"reranker_enabled"`
	RerankerBaseURL        string  `json:"reranker_base_url,omitempty"`
	DenseScoreThreshold    float64 `json:"dense_score_threshold,omitempty"`
	LexicalScoreThreshold  float64 `json:"lexical_score_threshold,omitempty"`
	FusedScoreThreshold    float64 `json:"fused_score_threshold,omitempty"`
	RerankerScoreThreshold float64 `json:"reranker_score_threshold,omitempty"`
}
