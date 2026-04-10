package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/logger"
	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	FeatureDB *sql.DB
)

func InitFeatureDB() error {
	if !conf.KnowledgeConfInfo.FeatureStoreEnabled() {
		return nil
	}

	db, err := sql.Open("pgx", conf.KnowledgeConfInfo.PostgresDSN)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	if err = initializeFeatureTables(db); err != nil {
		return err
	}

	if err = migrateFeatureAgentTables(db); err != nil {
		return err
	}

	FeatureDB = db
	logger.Info("feature db initialize successfully")
	return nil
}

func FeatureEnabled() bool {
	return FeatureDB != nil
}

func featureEmbeddingColumnType() string {
	dim := conf.KnowledgeConfInfo.EmbeddingDimensions
	if dim <= 0 {
		dim = 512
	}
	return fmt.Sprintf("vector(%d)", dim)
}

func initializeFeatureTables(db *sql.DB) error {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS vector;`,
		`CREATE EXTENSION IF NOT EXISTS pg_trgm;`,
		`
		CREATE TABLE IF NOT EXISTS agent_runs (
			id BIGSERIAL PRIMARY KEY,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			user_id VARCHAR(100) NOT NULL DEFAULT '',
			chat_id VARCHAR(255) NOT NULL DEFAULT '',
			msg_id VARCHAR(255) NOT NULL DEFAULT '',
			mode VARCHAR(100) NOT NULL DEFAULT '',
			input TEXT NOT NULL,
			final_output TEXT NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			token_total INTEGER NOT NULL DEFAULT 0,
			step_count INTEGER NOT NULL DEFAULT 0,
			replay_of BIGINT NOT NULL DEFAULT 0,
			skill_id VARCHAR(255) NOT NULL DEFAULT '',
			skill_name VARCHAR(255) NOT NULL DEFAULT '',
			skill_version VARCHAR(100) NOT NULL DEFAULT '',
			selector_reason TEXT NOT NULL DEFAULT '',
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_user_id ON agent_runs(user_id);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_workspace ON agent_runs(workspace_id, create_time);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_status ON agent_runs(status);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_mode ON agent_runs(mode);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_replay_of ON agent_runs(replay_of);
		CREATE INDEX IF NOT EXISTS idx_agent_runs_create_time ON agent_runs(create_time);
	`,
		`
		CREATE TABLE IF NOT EXISTS agent_steps (
			id BIGSERIAL PRIMARY KEY,
			workspace_id VARCHAR(100) NOT NULL DEFAULT 'default',
			run_id BIGINT NOT NULL,
			step_index INTEGER NOT NULL DEFAULT 0,
			kind VARCHAR(50) NOT NULL DEFAULT '',
			name VARCHAR(255) NOT NULL DEFAULT '',
			tool_name VARCHAR(255) NOT NULL DEFAULT '',
			skill_id VARCHAR(255) NOT NULL DEFAULT '',
			skill_name VARCHAR(255) NOT NULL DEFAULT '',
			skill_version VARCHAR(100) NOT NULL DEFAULT '',
			input TEXT NOT NULL DEFAULT '',
			raw_output TEXT NOT NULL DEFAULT '',
			observations JSONB NOT NULL DEFAULT '[]'::jsonb,
			allowed_tools JSONB NOT NULL DEFAULT '[]'::jsonb,
			step_context TEXT NOT NULL DEFAULT '',
			token INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(50) NOT NULL DEFAULT '',
			error TEXT NOT NULL DEFAULT '',
			provider VARCHAR(100) NOT NULL DEFAULT '',
			model VARCHAR(255) NOT NULL DEFAULT '',
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_agent_steps_workspace ON agent_steps(workspace_id, run_id);
		CREATE INDEX IF NOT EXISTS idx_agent_steps_run_id ON agent_steps(run_id);
		CREATE INDEX IF NOT EXISTS idx_agent_steps_run_idx ON agent_steps(run_id, step_index);
	`,
		`
		CREATE TABLE IF NOT EXISTS knowledge_bases (
			id BIGSERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT '',
			UNIQUE (from_bot, name)
		);
	`,
		`
		CREATE TABLE IF NOT EXISTS collections (
			id BIGSERIAL PRIMARY KEY,
			knowledge_base_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT '',
			UNIQUE (knowledge_base_id, name)
		);
		CREATE INDEX IF NOT EXISTS idx_collections_kb_id ON collections(knowledge_base_id);
	`,
		`
		CREATE TABLE IF NOT EXISTS documents (
			id BIGSERIAL PRIMARY KEY,
			collection_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			source_type VARCHAR(50) NOT NULL DEFAULT 'upload',
			content_type VARCHAR(100) NOT NULL DEFAULT '',
			object_key TEXT NOT NULL DEFAULT '',
			status VARCHAR(50) NOT NULL DEFAULT 'active',
			latest_version INTEGER NOT NULL DEFAULT 0,
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT '',
			UNIQUE (collection_id, name)
		);
		CREATE INDEX IF NOT EXISTS idx_documents_collection_id ON documents(collection_id);
	`,
		`
		CREATE TABLE IF NOT EXISTS document_versions (
			id BIGSERIAL PRIMARY KEY,
			document_id BIGINT NOT NULL,
			version INTEGER NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			file_md5 VARCHAR(255) NOT NULL DEFAULT '',
			object_key TEXT NOT NULL DEFAULT '',
			parsed_object_key TEXT NOT NULL DEFAULT '',
			file_size BIGINT NOT NULL DEFAULT 0,
			chunk_count INTEGER NOT NULL DEFAULT 0,
			error TEXT NOT NULL DEFAULT '',
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT '',
			UNIQUE (document_id, version)
		);
		CREATE INDEX IF NOT EXISTS idx_document_versions_document_id ON document_versions(document_id);
		CREATE INDEX IF NOT EXISTS idx_document_versions_status ON document_versions(status);
	`,
		`
		CREATE TABLE IF NOT EXISTS ingestion_jobs (
			id BIGSERIAL PRIMARY KEY,
			collection_id BIGINT NOT NULL,
			document_id BIGINT NOT NULL,
			document_version_id BIGINT NOT NULL,
			task_id VARCHAR(255) NOT NULL DEFAULT '',
			stage VARCHAR(100) NOT NULL DEFAULT 'pending',
			status VARCHAR(50) NOT NULL DEFAULT 'queued',
			error TEXT NOT NULL DEFAULT '',
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			start_time BIGINT NOT NULL DEFAULT 0,
			finish_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_document_id ON ingestion_jobs(document_id);
		CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_version_id ON ingestion_jobs(document_version_id);
		CREATE INDEX IF NOT EXISTS idx_ingestion_jobs_status ON ingestion_jobs(status);
	`,
		fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS chunks (
			id BIGSERIAL PRIMARY KEY,
			collection_id BIGINT NOT NULL,
			document_id BIGINT NOT NULL,
			document_version_id BIGINT NOT NULL,
			chunk_index INTEGER NOT NULL,
			content TEXT NOT NULL,
			content_lexical TEXT NOT NULL DEFAULT '',
			citation_label VARCHAR(255) NOT NULL DEFAULT '',
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			embedding %s,
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT '',
			UNIQUE (document_version_id, chunk_index)
		);
		CREATE INDEX IF NOT EXISTS idx_chunks_collection_id ON chunks(collection_id);
		CREATE INDEX IF NOT EXISTS idx_chunks_document_id ON chunks(document_id);
		CREATE INDEX IF NOT EXISTS idx_chunks_version_id ON chunks(document_version_id);
		CREATE INDEX IF NOT EXISTS idx_chunks_lexical ON chunks USING GIN (to_tsvector('simple', content_lexical));
		CREATE INDEX IF NOT EXISTS idx_chunks_content_trgm ON chunks USING GIN (content gin_trgm_ops);
	`, featureEmbeddingColumnType()),
		fmt.Sprintf(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_indexes WHERE schemaname = 'public' AND indexname = 'idx_chunks_embedding'
			) THEN
				EXECUTE 'CREATE INDEX idx_chunks_embedding ON chunks USING hnsw (embedding vector_cosine_ops)';
			END IF;
		END $$;
	`),
		`
		CREATE TABLE IF NOT EXISTS retrieval_runs (
			id BIGSERIAL PRIMARY KEY,
			knowledge_base_id BIGINT NOT NULL,
			collection_id BIGINT NOT NULL,
			query_text TEXT NOT NULL,
			query_normalized TEXT NOT NULL DEFAULT '',
			rewritten_query TEXT NOT NULL DEFAULT '',
			answer TEXT NOT NULL DEFAULT '',
			citations JSONB NOT NULL DEFAULT '[]'::jsonb,
			status VARCHAR(50) NOT NULL DEFAULT 'succeeded',
			error TEXT NOT NULL DEFAULT '',
			create_time BIGINT NOT NULL DEFAULT 0,
			update_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_retrieval_runs_collection_id ON retrieval_runs(collection_id);
		CREATE INDEX IF NOT EXISTS idx_retrieval_runs_create_time ON retrieval_runs(create_time);
	`,
		`
		CREATE TABLE IF NOT EXISTS retrieval_hits (
			id BIGSERIAL PRIMARY KEY,
			retrieval_run_id BIGINT NOT NULL,
			chunk_id BIGINT NOT NULL,
			document_id BIGINT NOT NULL,
			document_version_id BIGINT NOT NULL,
			rank_position INTEGER NOT NULL DEFAULT 0,
			dense_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			lexical_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			rrf_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			final_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			citation_label VARCHAR(255) NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
			create_time BIGINT NOT NULL DEFAULT 0,
			from_bot VARCHAR(255) NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_retrieval_hits_run_id ON retrieval_hits(retrieval_run_id);
	`,
	}

	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}

func migrateFeatureAgentTables(db *sql.DB) error {
	if db == nil {
		return nil
	}

	definitions := map[string]map[string]string{
		"agent_runs": {
			"workspace_id":    "VARCHAR(100) NOT NULL DEFAULT 'default'",
			"skill_id":        "VARCHAR(255) NOT NULL DEFAULT ''",
			"skill_name":      "VARCHAR(255) NOT NULL DEFAULT ''",
			"skill_version":   "VARCHAR(100) NOT NULL DEFAULT ''",
			"selector_reason": "TEXT NOT NULL DEFAULT ''",
		},
		"agent_steps": {
			"workspace_id":  "VARCHAR(100) NOT NULL DEFAULT 'default'",
			"skill_id":      "VARCHAR(255) NOT NULL DEFAULT ''",
			"skill_name":    "VARCHAR(255) NOT NULL DEFAULT ''",
			"skill_version": "VARCHAR(100) NOT NULL DEFAULT ''",
			"allowed_tools": "JSONB NOT NULL DEFAULT '[]'::jsonb",
			"step_context":  "TEXT NOT NULL DEFAULT ''",
		},
	}

	for tableName, columns := range definitions {
		for columnName, definition := range columns {
			if err := ensurePostgresColumn(db, tableName, columnName, definition); err != nil {
				return err
			}
		}
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_agent_runs_workspace ON agent_runs(workspace_id, create_time)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_steps_workspace ON agent_steps(workspace_id, run_id)`,
	}
	for _, statement := range indexes {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}

func ensurePostgresColumn(db *sql.DB, tableName, columnName, definition string) error {
	var count int
	query := `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`
	if err := db.QueryRow(query, tableName, columnName).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition))
	return err
}
