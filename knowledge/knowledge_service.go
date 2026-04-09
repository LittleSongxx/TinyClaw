package knowledge

import (
	"bytes"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/llm"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/langchaingo/documentloaders"
	"github.com/LittleSongxx/langchaingo/llms"
	"github.com/LittleSongxx/langchaingo/schema"
	"github.com/LittleSongxx/langchaingo/textsplitter"
	"github.com/hibiken/asynq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	ingestionTaskType = "knowledge:ingest"
	rrfK              = 60.0
)

type KnowledgeBase struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreateTime  int64  `json:"create_time"`
	UpdateTime  int64  `json:"update_time"`
}

type Collection struct {
	ID              int64  `json:"id"`
	KnowledgeBaseID int64  `json:"knowledge_base_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	CreateTime      int64  `json:"create_time"`
	UpdateTime      int64  `json:"update_time"`
}

type Document struct {
	ID             int64  `json:"id"`
	CollectionID   int64  `json:"collection_id"`
	Name           string `json:"name"`
	SourceType     string `json:"source_type"`
	ContentType    string `json:"content_type"`
	ObjectKey      string `json:"object_key"`
	Status         string `json:"status"`
	LatestVersion  int    `json:"latest_version"`
	CurrentStatus  string `json:"current_status,omitempty"`
	CurrentVersion int    `json:"current_version,omitempty"`
	ChunkCount     int    `json:"chunk_count,omitempty"`
	CreateTime     int64  `json:"create_time"`
	UpdateTime     int64  `json:"update_time"`
}

type DocumentVersion struct {
	ID              int64  `json:"id"`
	DocumentID      int64  `json:"document_id"`
	Version         int    `json:"version"`
	Status          string `json:"status"`
	FileMD5         string `json:"file_md5"`
	ObjectKey       string `json:"object_key"`
	ParsedObjectKey string `json:"parsed_object_key"`
	FileSize        int64  `json:"file_size"`
	ChunkCount      int    `json:"chunk_count"`
	Error           string `json:"error"`
	CreateTime      int64  `json:"create_time"`
	UpdateTime      int64  `json:"update_time"`
}

type IngestionJob struct {
	ID                int64  `json:"id"`
	CollectionID      int64  `json:"collection_id"`
	DocumentID        int64  `json:"document_id"`
	DocumentName      string `json:"document_name,omitempty"`
	DocumentVersionID int64  `json:"document_version_id"`
	Version           int    `json:"version,omitempty"`
	TaskID            string `json:"task_id"`
	Stage             string `json:"stage"`
	Status            string `json:"status"`
	Error             string `json:"error"`
	CreateTime        int64  `json:"create_time"`
	UpdateTime        int64  `json:"update_time"`
	StartTime         int64  `json:"start_time"`
	FinishTime        int64  `json:"finish_time"`
}

type RetrievalHit struct {
	ID                int64                  `json:"id"`
	ChunkID           int64                  `json:"chunk_id"`
	DocumentID        int64                  `json:"document_id"`
	DocumentVersionID int64                  `json:"document_version_id"`
	DocumentName      string                 `json:"document_name"`
	CitationLabel     string                 `json:"citation_label"`
	Content           string                 `json:"content"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	RankPosition      int                    `json:"rank_position"`
	DenseScore        float64                `json:"dense_score"`
	LexicalScore      float64                `json:"lexical_score"`
	RRFScore          float64                `json:"rrf_score"`
	RerankScore       float64                `json:"rerank_score,omitempty"`
	FinalScore        float64                `json:"final_score"`
	Reranked          bool                   `json:"-"`
}

type RetrievalRun struct {
	ID              int64    `json:"id"`
	KnowledgeBaseID int64    `json:"knowledge_base_id"`
	CollectionID    int64    `json:"collection_id"`
	QueryText       string   `json:"query_text"`
	QueryNormalized string   `json:"query_normalized"`
	RewrittenQuery  string   `json:"rewritten_query"`
	Answer          string   `json:"answer"`
	Citations       []string `json:"citations"`
	Status          string   `json:"status"`
	Error           string   `json:"error"`
	CreateTime      int64    `json:"create_time"`
	UpdateTime      int64    `json:"update_time"`
}

type RetrievalDebugResult struct {
	Run   *RetrievalRun  `json:"run"`
	Hits  []RetrievalHit `json:"hits"`
	Query string         `json:"query"`
}

type ListResult[T any] struct {
	List     []T `json:"list"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type Reranker interface {
	Rerank(ctx context.Context, query string, hits []RetrievalHit) ([]RetrievalHit, error)
}

type noopReranker struct{}

func (noopReranker) Rerank(_ context.Context, _ string, hits []RetrievalHit) ([]RetrievalHit, error) {
	return hits, nil
}

type KnowledgeService struct {
	db       *sql.DB
	minio    *minio.Client
	bucket   string
	queue    *asynq.Client
	server   *asynq.Server
	reranker Reranker
}

type ingestionPayload struct {
	JobID int64 `json:"job_id"`
}

var defaultService *KnowledgeService

func Enabled() bool {
	return defaultService != nil
}

func DefaultService() *KnowledgeService {
	return defaultService
}

func initKnowledge(ctx context.Context) error {
	if !conf.KnowledgeConfInfo.Enabled() {
		return nil
	}
	if db.FeatureDB == nil {
		return fmt.Errorf("feature db is not initialized")
	}

	minioClient, err := minio.New(conf.KnowledgeConfInfo.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(conf.KnowledgeConfInfo.MinIOAccessKey, conf.KnowledgeConfInfo.MinIOSecretKey, ""),
		Secure: conf.KnowledgeConfInfo.MinIOUseSSL,
	})
	if err != nil {
		return err
	}

	service := &KnowledgeService{
		db:       db.FeatureDB,
		minio:    minioClient,
		bucket:   conf.KnowledgeConfInfo.MinIOBucket,
		reranker: noopReranker{},
	}
	if strings.TrimSpace(conf.KnowledgeConfInfo.RerankerBaseURL) != "" {
		service.reranker = newTEIReranker(conf.KnowledgeConfInfo.RerankerBaseURL, conf.KnowledgeConfInfo.RerankerTopN)
	}
	if conf.KnowledgeConfInfo.QueueEnabled() {
		service.queue = asynq.NewClient(asynq.RedisClientOpt{
			Addr:     conf.KnowledgeConfInfo.RedisAddr,
			Password: conf.KnowledgeConfInfo.RedisPassword,
			DB:       conf.KnowledgeConfInfo.RedisDB,
		})
		service.server = asynq.NewServer(asynq.RedisClientOpt{
			Addr:     conf.KnowledgeConfInfo.RedisAddr,
			Password: conf.KnowledgeConfInfo.RedisPassword,
			DB:       conf.KnowledgeConfInfo.RedisDB,
		}, asynq.Config{Concurrency: 2})
	}

	exists, err := service.minio.BucketExists(ctx, service.bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err = service.minio.MakeBucket(ctx, service.bucket, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}

	if _, _, err = service.ensureDefaultCollection(ctx); err != nil {
		return err
	}

	if service.server != nil {
		mux := asynq.NewServeMux()
		mux.HandleFunc(ingestionTaskType, service.handleIngestionTask)
		go func() {
			if runErr := service.server.Run(mux); runErr != nil {
				logger.Error("knowledge worker stopped", "err", runErr)
			}
		}()
	}

	defaultService = service

	if conf.KnowledgeConfInfo.KnowledgeAutoMigrate {
		go func() {
			migrateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
			defer cancel()
			if err := service.autoMigrateLegacyKnowledge(migrateCtx); err != nil {
				logger.Error("knowledge auto migrate fail", "err", err)
			}
		}()
	}

	return nil
}

func (s *KnowledgeService) ensureDefaultCollection(ctx context.Context) (*KnowledgeBase, *Collection, error) {
	kb, err := s.getOrCreateKnowledgeBase(ctx, conf.KnowledgeConfInfo.KnowledgeBaseName(), "")
	if err != nil {
		return nil, nil, err
	}

	collection, err := s.getOrCreateCollection(ctx, kb.ID, conf.KnowledgeConfInfo.CollectionName(), "")
	if err != nil {
		return nil, nil, err
	}

	return kb, collection, nil
}

func (s *KnowledgeService) getOrCreateKnowledgeBase(ctx context.Context, name, description string) (*KnowledgeBase, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, description, status, create_time, update_time
		FROM knowledge_bases WHERE from_bot = $1 AND name = $2`,
		conf.BaseConfInfo.BotName, name,
	)

	var kb KnowledgeBase
	if err := row.Scan(&kb.ID, &kb.Name, &kb.Description, &kb.Status, &kb.CreateTime, &kb.UpdateTime); err == nil {
		return &kb, nil
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	now := time.Now().Unix()
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO knowledge_bases (name, description, status, create_time, update_time, from_bot)
		VALUES ($1, $2, 'active', $3, $4, $5) RETURNING id`,
		name, description, now, now, conf.BaseConfInfo.BotName,
	).Scan(&kb.ID)
	if err != nil {
		return nil, err
	}

	kb.Name = name
	kb.Description = description
	kb.Status = "active"
	kb.CreateTime = now
	kb.UpdateTime = now
	return &kb, nil
}

func (s *KnowledgeService) getOrCreateCollection(ctx context.Context, knowledgeBaseID int64, name, description string) (*Collection, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, knowledge_base_id, name, description, status, create_time, update_time
		FROM collections WHERE knowledge_base_id = $1 AND name = $2`,
		knowledgeBaseID, name,
	)

	var collection Collection
	if err := row.Scan(&collection.ID, &collection.KnowledgeBaseID, &collection.Name, &collection.Description, &collection.Status, &collection.CreateTime, &collection.UpdateTime); err == nil {
		return &collection, nil
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	now := time.Now().Unix()
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO collections (knowledge_base_id, name, description, status, create_time, update_time, from_bot)
		VALUES ($1, $2, $3, 'active', $4, $5, $6) RETURNING id`,
		knowledgeBaseID, name, description, now, now, conf.BaseConfInfo.BotName,
	).Scan(&collection.ID)
	if err != nil {
		return nil, err
	}

	collection.KnowledgeBaseID = knowledgeBaseID
	collection.Name = name
	collection.Description = description
	collection.Status = "active"
	collection.CreateTime = now
	collection.UpdateTime = now
	return &collection, nil
}

func (s *KnowledgeService) listCollections(ctx context.Context) (*ListResult[Collection], error) {
	kb, _, err := s.ensureDefaultCollection(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, knowledge_base_id, name, description, status, create_time, update_time
		FROM collections WHERE knowledge_base_id = $1 ORDER BY id ASC`,
		kb.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	collections := make([]Collection, 0)
	for rows.Next() {
		var collection Collection
		if err = rows.Scan(&collection.ID, &collection.KnowledgeBaseID, &collection.Name, &collection.Description, &collection.Status, &collection.CreateTime, &collection.UpdateTime); err != nil {
			return nil, err
		}
		collections = append(collections, collection)
	}

	return &ListResult[Collection]{List: collections, Total: len(collections), Page: 1, PageSize: len(collections)}, rows.Err()
}

func (s *KnowledgeService) upsertDocumentContent(ctx context.Context, collectionName, name, sourceType, contentType string, data []byte) (*Document, *DocumentVersion, *IngestionJob, error) {
	kb, collection, err := s.ensureDefaultCollection(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	if strings.TrimSpace(collectionName) != "" && collectionName != collection.Name {
		collection, err = s.getOrCreateCollection(ctx, kb.ID, collectionName, "")
		if err != nil {
			return nil, nil, nil, err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	defer tx.Rollback()

	document, err := s.getDocumentByNameTx(ctx, tx, collection.ID, name)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, nil, err
	}

	now := time.Now().Unix()
	objectKey := s.sourceObjectKey(collection.ID, name, now)
	md5Text := checksumMD5(data)

	reader := bytes.NewReader(data)
	_, err = s.minio.PutObject(ctx, s.bucket, objectKey, reader, int64(len(data)), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, nil, nil, err
	}

	if document == nil {
		document = &Document{}
		err = tx.QueryRowContext(ctx,
			`INSERT INTO documents (collection_id, name, source_type, content_type, object_key, status, latest_version, create_time, update_time, from_bot)
			VALUES ($1, $2, $3, $4, $5, 'active', 0, $6, $7, $8) RETURNING id`,
			collection.ID, name, sourceType, contentType, objectKey, now, now, conf.BaseConfInfo.BotName,
		).Scan(&document.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		document.CollectionID = collection.ID
		document.Name = name
		document.SourceType = sourceType
		document.ContentType = contentType
		document.ObjectKey = objectKey
		document.Status = "active"
		document.LatestVersion = 0
		document.CreateTime = now
		document.UpdateTime = now
	} else {
		_, err = tx.ExecContext(ctx,
			`UPDATE documents SET source_type = $1, content_type = $2, object_key = $3, update_time = $4 WHERE id = $5`,
			sourceType, contentType, objectKey, now, document.ID,
		)
		if err != nil {
			return nil, nil, nil, err
		}
		document.SourceType = sourceType
		document.ContentType = contentType
		document.ObjectKey = objectKey
		document.UpdateTime = now
	}

	nextVersion := document.LatestVersion + 1
	if currentVersion, err := s.getMaxDocumentVersionTx(ctx, tx, document.ID); err == nil && currentVersion >= nextVersion {
		nextVersion = currentVersion + 1
	}

	version := &DocumentVersion{
		DocumentID: document.ID,
		Version:    nextVersion,
		Status:     "pending",
		FileMD5:    md5Text,
		ObjectKey:  objectKey,
		FileSize:   int64(len(data)),
		CreateTime: now,
		UpdateTime: now,
	}
	err = tx.QueryRowContext(ctx,
		`INSERT INTO document_versions (document_id, version, status, file_md5, object_key, file_size, chunk_count, error, create_time, update_time, from_bot)
		VALUES ($1, $2, 'pending', $3, $4, $5, 0, '', $6, $7, $8) RETURNING id`,
		document.ID, nextVersion, md5Text, objectKey, len(data), now, now, conf.BaseConfInfo.BotName,
	).Scan(&version.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	job := &IngestionJob{
		CollectionID:      collection.ID,
		DocumentID:        document.ID,
		DocumentVersionID: version.ID,
		Stage:             "pending",
		Status:            "queued",
		CreateTime:        now,
		UpdateTime:        now,
	}
	err = tx.QueryRowContext(ctx,
		`INSERT INTO ingestion_jobs (collection_id, document_id, document_version_id, task_id, stage, status, error, create_time, update_time, start_time, finish_time, from_bot)
		VALUES ($1, $2, $3, '', 'pending', 'queued', '', $4, $5, 0, 0, $6) RETURNING id`,
		collection.ID, document.ID, version.ID, now, now, conf.BaseConfInfo.BotName,
	).Scan(&job.ID)
	if err != nil {
		return nil, nil, nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, nil, nil, err
	}

	if s.queue != nil {
		payloadBody, err := json.Marshal(ingestionPayload{JobID: job.ID})
		if err != nil {
			return nil, nil, nil, err
		}

		info, err := s.queue.EnqueueContext(ctx, asynq.NewTask(ingestionTaskType, payloadBody))
		if err != nil {
			return nil, nil, nil, err
		}

		job.TaskID = info.ID
		_, err = s.db.ExecContext(ctx, `UPDATE ingestion_jobs SET task_id = $1, update_time = $2 WHERE id = $3`, info.ID, time.Now().Unix(), job.ID)
		if err != nil {
			return nil, nil, nil, err
		}
		return document, version, job, nil
	}

	if err = s.processIngestionJob(ctx, job.ID); err != nil {
		job.Status = "failed"
		job.Error = err.Error()
		return document, version, job, err
	}
	job.Status = "succeeded"
	job.Stage = "finished"
	job.UpdateTime = time.Now().Unix()
	return document, version, job, nil
}

func (s *KnowledgeService) getDocumentByNameTx(ctx context.Context, tx *sql.Tx, collectionID int64, name string) (*Document, error) {
	row := tx.QueryRowContext(ctx,
		`SELECT id, collection_id, name, source_type, content_type, object_key, status, latest_version, create_time, update_time
		FROM documents WHERE collection_id = $1 AND name = $2`,
		collectionID, name,
	)
	var document Document
	if err := row.Scan(&document.ID, &document.CollectionID, &document.Name, &document.SourceType, &document.ContentType, &document.ObjectKey, &document.Status, &document.LatestVersion, &document.CreateTime, &document.UpdateTime); err != nil {
		return nil, err
	}
	return &document, nil
}

func (s *KnowledgeService) getDocumentByName(ctx context.Context, collectionID int64, name string) (*Document, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, collection_id, name, source_type, content_type, object_key, status, latest_version, create_time, update_time
		FROM documents WHERE collection_id = $1 AND name = $2`,
		collectionID, name,
	)
	var document Document
	if err := row.Scan(&document.ID, &document.CollectionID, &document.Name, &document.SourceType, &document.ContentType, &document.ObjectKey, &document.Status, &document.LatestVersion, &document.CreateTime, &document.UpdateTime); err != nil {
		return nil, err
	}
	return &document, nil
}

func (s *KnowledgeService) getMaxDocumentVersionTx(ctx context.Context, tx *sql.Tx, documentID int64) (int, error) {
	var version sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT MAX(version) FROM document_versions WHERE document_id = $1`, documentID).Scan(&version); err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}

func (s *KnowledgeService) sourceObjectKey(collectionID int64, name string, ts int64) string {
	return fmt.Sprintf("%s/%s/%d/%d-%s", conf.BaseConfInfo.BotName, conf.KnowledgeConfInfo.CollectionName(), collectionID, ts, filepath.Base(name))
}

func (s *KnowledgeService) parsedObjectKey(documentID int64, version int) string {
	return fmt.Sprintf("%s/%s/%d/v%d/parsed.json", conf.BaseConfInfo.BotName, conf.KnowledgeConfInfo.CollectionName(), documentID, version)
}

func (s *KnowledgeService) handleIngestionTask(ctx context.Context, task *asynq.Task) error {
	payload := ingestionPayload{}
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	return s.processIngestionJob(ctx, payload.JobID)
}

func (s *KnowledgeService) processIngestionJob(ctx context.Context, jobID int64) error {
	job, version, document, collection, err := s.loadJobBundle(ctx, jobID)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	if _, err = s.db.ExecContext(ctx, `UPDATE ingestion_jobs SET status = 'processing', stage = 'download', start_time = $1, update_time = $1 WHERE id = $2`, now, job.ID); err != nil {
		return err
	}
	if _, err = s.db.ExecContext(ctx, `UPDATE document_versions SET status = 'processing', update_time = $1 WHERE id = $2`, now, version.ID); err != nil {
		return err
	}

	object, err := s.minio.GetObject(ctx, s.bucket, version.ObjectKey, minio.GetObjectOptions{})
	if err != nil {
		return s.failJob(ctx, jobID, version.ID, "download", err)
	}
	defer object.Close()

	body, err := io.ReadAll(object)
	if err != nil {
		return s.failJob(ctx, jobID, version.ID, "download", err)
	}

	if _, err = s.db.ExecContext(ctx, `UPDATE ingestion_jobs SET stage = 'parse', update_time = $1 WHERE id = $2`, time.Now().Unix(), job.ID); err != nil {
		return err
	}

	docs, err := loadDocumentsFromBytes(ctx, document.Name, version.FileMD5, body)
	if err != nil {
		return s.failJob(ctx, jobID, version.ID, "parse", err)
	}

	if _, err = s.db.ExecContext(ctx, `UPDATE ingestion_jobs SET stage = 'embed', update_time = $1 WHERE id = $2`, time.Now().Unix(), job.ID); err != nil {
		return err
	}

	texts := make([]string, 0, len(docs))
	for _, doc := range docs {
		texts = append(texts, doc.PageContent)
	}

	embeddingsRes, err := conf.KnowledgeConfInfo.Embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return s.failJob(ctx, jobID, version.ID, "embed", err)
	}
	if len(embeddingsRes) != len(docs) {
		return s.failJob(ctx, jobID, version.ID, "embed", fmt.Errorf("embedding count mismatch"))
	}

	if _, err = s.db.ExecContext(ctx, `UPDATE ingestion_jobs SET stage = 'index', update_time = $1 WHERE id = $2`, time.Now().Unix(), job.ID); err != nil {
		return err
	}

	if err = s.indexVersionChunks(ctx, collection, document, version, docs, embeddingsRes); err != nil {
		return s.failJob(ctx, jobID, version.ID, "index", err)
	}

	parsedKey := s.parsedObjectKey(document.ID, version.Version)
	parsedBody, _ := json.Marshal(docs)
	_, err = s.minio.PutObject(ctx, s.bucket, parsedKey, bytes.NewReader(parsedBody), int64(len(parsedBody)), minio.PutObjectOptions{ContentType: "application/json"})
	if err != nil {
		return s.failJob(ctx, jobID, version.ID, "index", err)
	}

	finishTime := time.Now().Unix()
	_, err = s.db.ExecContext(ctx,
		`UPDATE document_versions SET status = 'ready', parsed_object_key = $1, chunk_count = $2, error = '', update_time = $3 WHERE id = $4`,
		parsedKey, len(docs), finishTime, version.ID,
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE documents SET latest_version = $1, update_time = $2 WHERE id = $3`,
		version.Version, finishTime, document.ID,
	)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE ingestion_jobs SET status = 'succeeded', stage = 'done', error = '', finish_time = $1, update_time = $1 WHERE id = $2`,
		finishTime, job.ID,
	)
	return err
}

func (s *KnowledgeService) failJob(ctx context.Context, jobID, versionID int64, stage string, err error) error {
	now := time.Now().Unix()
	if _, execErr := s.db.ExecContext(ctx,
		`UPDATE ingestion_jobs SET status = 'failed', stage = $1, error = $2, finish_time = $3, update_time = $3 WHERE id = $4`,
		stage, err.Error(), now, jobID,
	); execErr != nil {
		return execErr
	}
	if _, execErr := s.db.ExecContext(ctx,
		`UPDATE document_versions SET status = 'failed', error = $1, update_time = $2 WHERE id = $3`,
		err.Error(), now, versionID,
	); execErr != nil {
		return execErr
	}
	return err
}

func (s *KnowledgeService) loadJobBundle(ctx context.Context, jobID int64) (*IngestionJob, *DocumentVersion, *Document, *Collection, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT j.id, j.collection_id, j.document_id, j.document_version_id, j.task_id, j.stage, j.status, j.error, j.create_time, j.update_time, j.start_time, j.finish_time,
		        d.name, dv.version, dv.status, dv.file_md5, dv.object_key, dv.parsed_object_key, dv.file_size, dv.chunk_count, dv.error,
		        c.id, c.knowledge_base_id, c.name, c.description, c.status, c.create_time, c.update_time
		FROM ingestion_jobs j
		JOIN documents d ON d.id = j.document_id
		JOIN document_versions dv ON dv.id = j.document_version_id
		JOIN collections c ON c.id = j.collection_id
		WHERE j.id = $1`,
		jobID,
	)

	job := &IngestionJob{}
	version := &DocumentVersion{}
	document := &Document{}
	collection := &Collection{}
	err := row.Scan(
		&job.ID, &job.CollectionID, &job.DocumentID, &job.DocumentVersionID, &job.TaskID, &job.Stage, &job.Status, &job.Error, &job.CreateTime, &job.UpdateTime, &job.StartTime, &job.FinishTime,
		&document.Name, &version.Version, &version.Status, &version.FileMD5, &version.ObjectKey, &version.ParsedObjectKey, &version.FileSize, &version.ChunkCount, &version.Error,
		&collection.ID, &collection.KnowledgeBaseID, &collection.Name, &collection.Description, &collection.Status, &collection.CreateTime, &collection.UpdateTime,
	)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	document.ID = job.DocumentID
	document.CollectionID = job.CollectionID
	version.ID = job.DocumentVersionID
	version.DocumentID = job.DocumentID

	return job, version, document, collection, nil
}

func (s *KnowledgeService) indexVersionChunks(ctx context.Context, collection *Collection, document *Document, version *DocumentVersion, docs []schema.Document, embeddingsRes [][]float32) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `DELETE FROM chunks WHERE document_version_id = $1`, version.ID)
	if err != nil {
		return err
	}

	now := time.Now().Unix()
	for index, docItem := range docs {
		metadata := docItem.Metadata
		if metadata == nil {
			metadata = map[string]interface{}{}
		}
		metadata["file_name"] = document.Name
		metadata["file_md5"] = version.FileMD5
		metadata["chunk_index"] = index

		metadataBody, err := json.Marshal(metadata)
		if err != nil {
			return err
		}

		citationLabel := fmt.Sprintf("[D%d] %s#%d", index+1, document.Name, index+1)
		_, err = tx.ExecContext(ctx,
			`INSERT INTO chunks (collection_id, document_id, document_version_id, chunk_index, content, content_lexical, citation_label, metadata, embedding, create_time, update_time, from_bot)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, CAST($9 AS vector), $10, $11, $12)`,
			collection.ID, document.ID, version.ID, index, docItem.PageContent, normalizeLexicalText(docItem.PageContent), citationLabel, string(metadataBody), vectorLiteral(embeddingsRes[index]), now, now, conf.BaseConfInfo.BotName,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *KnowledgeService) listDocuments(ctx context.Context, page, pageSize int, name string) (*ListResult[Document], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	_, collection, err := s.ensureDefaultCollection(ctx)
	if err != nil {
		return nil, err
	}

	args := []interface{}{collection.ID}
	where := `WHERE d.collection_id = $1 AND d.status != 'deleted'`
	if strings.TrimSpace(name) != "" {
		where += " AND d.name ILIKE $2"
		args = append(args, "%"+strings.TrimSpace(name)+"%")
	}
	countQuery := "SELECT COUNT(*) FROM documents d " + where

	var total int
	if err = s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	args = append(args, pageSize, (page-1)*pageSize)
	query := `
		SELECT d.id, d.collection_id, d.name, d.source_type, d.content_type, d.object_key, d.status, d.latest_version, d.create_time, d.update_time,
		       COALESCE(dv.status, 'pending') AS current_status,
		       COALESCE(dv.version, 0) AS current_version,
		       COALESCE(dv.chunk_count, 0) AS chunk_count
		FROM documents d
		LEFT JOIN LATERAL (
			SELECT version, status, chunk_count
			FROM document_versions
			WHERE document_id = d.id
			ORDER BY version DESC
			LIMIT 1
		) dv ON true ` + where + `
		ORDER BY d.id DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]Document, 0)
	for rows.Next() {
		var document Document
		if err = rows.Scan(&document.ID, &document.CollectionID, &document.Name, &document.SourceType, &document.ContentType, &document.ObjectKey, &document.Status, &document.LatestVersion,
			&document.CreateTime, &document.UpdateTime, &document.CurrentStatus, &document.CurrentVersion, &document.ChunkCount); err != nil {
			return nil, err
		}
		result = append(result, document)
	}

	return &ListResult[Document]{List: result, Total: total, Page: page, PageSize: pageSize}, rows.Err()
}

func (s *KnowledgeService) listIngestionJobs(ctx context.Context, page, pageSize int, status string) (*ListResult[IngestionJob], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	where := "WHERE j.from_bot = $1"
	args := []interface{}{conf.BaseConfInfo.BotName}
	if strings.TrimSpace(status) != "" {
		where += " AND j.status = $2"
		args = append(args, status)
	}

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ingestion_jobs j "+where, args...).Scan(&total); err != nil {
		return nil, err
	}

	args = append(args, pageSize, (page-1)*pageSize)
	query := `
		SELECT j.id, j.collection_id, j.document_id, d.name, j.document_version_id, dv.version, j.task_id, j.stage, j.status, j.error,
		       j.create_time, j.update_time, j.start_time, j.finish_time
		FROM ingestion_jobs j
		JOIN documents d ON d.id = j.document_id
		JOIN document_versions dv ON dv.id = j.document_version_id ` + where + `
		ORDER BY j.id DESC
		LIMIT $` + fmt.Sprintf("%d", len(args)-1) + ` OFFSET $` + fmt.Sprintf("%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]IngestionJob, 0)
	for rows.Next() {
		var job IngestionJob
		if err = rows.Scan(&job.ID, &job.CollectionID, &job.DocumentID, &job.DocumentName, &job.DocumentVersionID, &job.Version, &job.TaskID, &job.Stage, &job.Status, &job.Error,
			&job.CreateTime, &job.UpdateTime, &job.StartTime, &job.FinishTime); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}

	return &ListResult[IngestionJob]{List: jobs, Total: total, Page: page, PageSize: pageSize}, rows.Err()
}

func (s *KnowledgeService) getDocumentContent(ctx context.Context, name string) (string, error) {
	_, collection, err := s.ensureDefaultCollection(ctx)
	if err != nil {
		return "", err
	}

	document, err := s.getDocumentByName(ctx, collection.ID, name)
	if err != nil {
		return "", err
	}

	var objectKey string
	err = s.db.QueryRowContext(ctx,
		`SELECT object_key FROM document_versions WHERE document_id = $1 ORDER BY version DESC LIMIT 1`,
		document.ID,
	).Scan(&objectKey)
	if err != nil {
		return "", err
	}

	object, err := s.minio.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return "", err
	}
	defer object.Close()

	body, err := io.ReadAll(object)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *KnowledgeService) deleteDocumentByName(ctx context.Context, name string) error {
	_, collection, err := s.ensureDefaultCollection(ctx)
	if err != nil {
		return err
	}

	document, err := s.getDocumentByName(ctx, collection.ID, name)
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	if _, err = tx.ExecContext(ctx, `UPDATE documents SET status = 'deleted', update_time = $1 WHERE id = $2`, now, document.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE document_versions SET status = 'deleted', update_time = $1 WHERE document_id = $2`, now, document.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM chunks WHERE document_id = $1`, document.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *KnowledgeService) autoMigrateLegacyKnowledge(ctx context.Context) error {
	entries, err := os.ReadDir(conf.KnowledgeConfInfo.KnowledgePath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".txt" && ext != ".pdf" && ext != ".csv" && ext != ".html" {
			continue
		}

		fullPath := filepath.Join(conf.KnowledgeConfInfo.KnowledgePath, entry.Name())
		body, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		_, collection, err := s.ensureDefaultCollection(ctx)
		if err != nil {
			return err
		}
		document, err := s.getDocumentByName(ctx, collection.ID, entry.Name())
		if err == nil {
			contentMD5 := checksumMD5(body)
			var latestMD5 string
			scanErr := s.db.QueryRowContext(ctx,
				`SELECT file_md5 FROM document_versions WHERE document_id = $1 ORDER BY version DESC LIMIT 1`,
				document.ID,
			).Scan(&latestMD5)
			if scanErr == nil && latestMD5 == contentMD5 {
				continue
			}
		} else if err != sql.ErrNoRows {
			return err
		}

		if _, _, _, err = s.upsertDocumentContent(ctx, conf.KnowledgeConfInfo.CollectionName(), entry.Name(), "migration", contentTypeFromName(entry.Name()), body); err != nil {
			return err
		}
	}

	return nil
}

func (s *KnowledgeService) debugRetrieve(ctx context.Context, query string, persist bool) (*RetrievalDebugResult, error) {
	kb, collection, err := s.ensureDefaultCollection(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.TrimSpace(query)
	queryNormalized := normalizeLexicalText(query)
	rewrittenQuery := query
	queryEmbedding, err := conf.KnowledgeConfInfo.Embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}

	denseHits, err := s.searchDense(ctx, collection.ID, queryEmbedding, 12)
	if err != nil {
		return nil, err
	}
	lexicalHits, err := s.searchLexical(ctx, collection.ID, queryNormalized, query, 12)
	if err != nil {
		return nil, err
	}

	hits := fuseRetrievalHits(denseHits, lexicalHits)
	hits, err = s.reranker.Rerank(ctx, rewrittenQuery, hits)
	if err != nil {
		logger.WarnCtx(ctx, "knowledge rerank failed; falling back to fused hits", "err", err)
		hits = fuseRetrievalHits(denseHits, lexicalHits)
	}
	hits = filterRetrievalHits(hits, currentRetrievalThresholds())

	for i := range hits {
		hits[i].RankPosition = i + 1
		if hits[i].FinalScore == 0 {
			hits[i].FinalScore = hits[i].RRFScore
		}
	}

	status := "succeeded"
	if len(hits) == 0 {
		status = "no_match"
	}
	run := &RetrievalRun{
		KnowledgeBaseID: kb.ID,
		CollectionID:    collection.ID,
		QueryText:       query,
		QueryNormalized: queryNormalized,
		RewrittenQuery:  rewrittenQuery,
		Status:          status,
		Citations:       hitCitations(hits),
		CreateTime:      time.Now().Unix(),
		UpdateTime:      time.Now().Unix(),
	}
	if persist {
		if err = s.insertRetrievalRun(ctx, run, hits); err != nil {
			return nil, err
		}
	}

	return &RetrievalDebugResult{
		Run:   run,
		Hits:  hits,
		Query: query,
	}, nil
}

func (s *KnowledgeService) answerWithLLM(ctx context.Context, callLLM *llm.LLM, userInput string) (string, *RetrievalDebugResult, error) {
	debugResult, err := s.debugRetrieve(ctx, userInput, true)
	if err != nil {
		return "", nil, err
	}
	if len(debugResult.Hits) == 0 {
		answer := noRelevantKnowledgeAnswer()
		callLLM.WholeContent = answer
		callLLM.DirectSendMsg(answer, true)
		callLLM.Content = userInput
		debugResult.Run.Status = "no_match"
		if err = callLLM.InsertOrUpdate(); err != nil {
			return "", nil, err
		}
		debugResult.Run.Answer = answer
		if _, err = s.db.ExecContext(ctx, `UPDATE retrieval_runs SET answer = $1, citations = $2::jsonb, status = $3, update_time = $4 WHERE id = $5`,
			answer, mustJSON([]string{}), "no_match", time.Now().Unix(), debugResult.Run.ID); err != nil {
			return "", nil, err
		}
		return answer, debugResult, nil
	}

	prompt := buildKnowledgePrompt(userInput, debugResult.Hits)
	callLLM.PrepareRuntimeTools()
	callLLM.GetMessages(callLLM.UserId, prompt)
	callLLM.Content = prompt
	callLLM.LLMClient.GetModel(callLLM)

	if callLLM.MessageChan != nil || callLLM.HTTPMsgChan != nil {
		err = callLLM.LLMClient.Send(ctx, callLLM)
	} else {
		callLLM.WholeContent, err = callLLM.LLMClient.SyncSend(ctx, callLLM)
	}
	if err != nil {
		return "", nil, err
	}

	answer := strings.TrimSpace(callLLM.WholeContent)
	if len(debugResult.Hits) > 0 && !containsCitation(answer) {
		answer = strings.TrimSpace(answer + "\n\nReferences:\n" + strings.Join(hitCitations(debugResult.Hits), "\n"))
		callLLM.WholeContent = answer
	}

	callLLM.Content = userInput
	if err = callLLM.InsertOrUpdate(); err != nil {
		return "", nil, err
	}

	debugResult.Run.Answer = answer
	if _, err = s.db.ExecContext(ctx, `UPDATE retrieval_runs SET answer = $1, citations = $2::jsonb, update_time = $3 WHERE id = $4`,
		answer, mustJSON(hitCitations(debugResult.Hits)), time.Now().Unix(), debugResult.Run.ID); err != nil {
		return "", nil, err
	}

	return answer, debugResult, nil
}

func (s *KnowledgeService) insertRetrievalRun(ctx context.Context, run *RetrievalRun, hits []RetrievalHit) error {
	if run == nil {
		return fmt.Errorf("retrieval run is nil")
	}

	now := time.Now().Unix()
	run.CreateTime = now
	run.UpdateTime = now
	citationsJSON, err := json.Marshal(run.Citations)
	if err != nil {
		return err
	}

	err = s.db.QueryRowContext(ctx,
		`INSERT INTO retrieval_runs (knowledge_base_id, collection_id, query_text, query_normalized, rewritten_query, answer, citations, status, error, create_time, update_time, from_bot)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12) RETURNING id`,
		run.KnowledgeBaseID, run.CollectionID, run.QueryText, run.QueryNormalized, run.RewrittenQuery, run.Answer, string(citationsJSON), run.Status, run.Error, now, now, conf.BaseConfInfo.BotName,
	).Scan(&run.ID)
	if err != nil {
		return err
	}

	for _, hit := range hits {
		metadataJSON, jsonErr := json.Marshal(hit.Metadata)
		if jsonErr != nil {
			return jsonErr
		}
		_, err = s.db.ExecContext(ctx,
			`INSERT INTO retrieval_hits (retrieval_run_id, chunk_id, document_id, document_version_id, rank_position, dense_score, lexical_score, rrf_score, final_score, citation_label, content, metadata, create_time, from_bot)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13, $14)`,
			run.ID, hit.ChunkID, hit.DocumentID, hit.DocumentVersionID, hit.RankPosition, hit.DenseScore, hit.LexicalScore, hit.RRFScore, hit.FinalScore, hit.CitationLabel, hit.Content, string(metadataJSON), now, conf.BaseConfInfo.BotName,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *KnowledgeService) listRetrievalRuns(ctx context.Context, page, pageSize int) (*ListResult[RetrievalRun], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM retrieval_runs WHERE from_bot = $1`, conf.BaseConfInfo.BotName).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, knowledge_base_id, collection_id, query_text, query_normalized, rewritten_query, answer, citations, status, error, create_time, update_time
		FROM retrieval_runs WHERE from_bot = $1 ORDER BY id DESC LIMIT $2 OFFSET $3`,
		conf.BaseConfInfo.BotName, pageSize, (page-1)*pageSize,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]RetrievalRun, 0)
	for rows.Next() {
		var (
			run          RetrievalRun
			citationsRaw string
		)
		if err = rows.Scan(&run.ID, &run.KnowledgeBaseID, &run.CollectionID, &run.QueryText, &run.QueryNormalized, &run.RewrittenQuery, &run.Answer, &citationsRaw, &run.Status, &run.Error, &run.CreateTime, &run.UpdateTime); err != nil {
			return nil, err
		}
		if citationsRaw != "" {
			_ = json.Unmarshal([]byte(citationsRaw), &run.Citations)
		}
		runs = append(runs, run)
	}

	return &ListResult[RetrievalRun]{List: runs, Total: total, Page: page, PageSize: pageSize}, rows.Err()
}

func (s *KnowledgeService) getRetrievalRun(ctx context.Context, id int64) (*RetrievalDebugResult, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, knowledge_base_id, collection_id, query_text, query_normalized, rewritten_query, answer, citations, status, error, create_time, update_time
		FROM retrieval_runs WHERE id = $1 AND from_bot = $2`,
		id, conf.BaseConfInfo.BotName,
	)

	var (
		run          RetrievalRun
		citationsRaw string
	)
	if err := row.Scan(&run.ID, &run.KnowledgeBaseID, &run.CollectionID, &run.QueryText, &run.QueryNormalized, &run.RewrittenQuery, &run.Answer, &citationsRaw, &run.Status, &run.Error, &run.CreateTime, &run.UpdateTime); err != nil {
		return nil, err
	}
	if citationsRaw != "" {
		_ = json.Unmarshal([]byte(citationsRaw), &run.Citations)
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT chunk_id, document_id, document_version_id, rank_position, dense_score, lexical_score, rrf_score, final_score, citation_label, content, metadata
		FROM retrieval_hits WHERE retrieval_run_id = $1 ORDER BY rank_position ASC`,
		run.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]RetrievalHit, 0)
	for rows.Next() {
		var (
			hit         RetrievalHit
			metadataRaw string
		)
		if err = rows.Scan(&hit.ChunkID, &hit.DocumentID, &hit.DocumentVersionID, &hit.RankPosition, &hit.DenseScore, &hit.LexicalScore, &hit.RRFScore, &hit.FinalScore, &hit.CitationLabel, &hit.Content, &metadataRaw); err != nil {
			return nil, err
		}
		if metadataRaw != "" {
			_ = json.Unmarshal([]byte(metadataRaw), &hit.Metadata)
		}
		hits = append(hits, hit)
	}

	return &RetrievalDebugResult{Run: &run, Hits: hits, Query: run.QueryText}, rows.Err()
}

func (s *KnowledgeService) searchDense(ctx context.Context, collectionID int64, embedding []float32, limit int) ([]RetrievalHit, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.document_id, c.document_version_id, d.name, c.citation_label, c.content, c.metadata,
		        1 - (c.embedding <=> CAST($1 AS vector)) AS dense_score
		FROM chunks c
		JOIN documents d ON d.id = c.document_id
		JOIN document_versions dv ON dv.id = c.document_version_id
		WHERE c.collection_id = $2 AND d.status = 'active' AND dv.status = 'ready' AND d.latest_version = dv.version
		ORDER BY c.embedding <=> CAST($1 AS vector)
		LIMIT $3`,
		vectorLiteral(embedding), collectionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]RetrievalHit, 0)
	for rows.Next() {
		var (
			hit         RetrievalHit
			metadataRaw string
		)
		if err = rows.Scan(&hit.ChunkID, &hit.DocumentID, &hit.DocumentVersionID, &hit.DocumentName, &hit.CitationLabel, &hit.Content, &metadataRaw, &hit.DenseScore); err != nil {
			return nil, err
		}
		if metadataRaw != "" {
			_ = json.Unmarshal([]byte(metadataRaw), &hit.Metadata)
		}
		hits = append(hits, hit)
	}

	return hits, rows.Err()
}

func (s *KnowledgeService) searchLexical(ctx context.Context, collectionID int64, normalizedQuery, rawQuery string, limit int) ([]RetrievalHit, error) {
	if strings.TrimSpace(normalizedQuery) == "" {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT c.id, c.document_id, c.document_version_id, d.name, c.citation_label, c.content, c.metadata,
		        ts_rank(to_tsvector('simple', c.content_lexical), plainto_tsquery('simple', $1)) AS lexical_score
		FROM chunks c
		JOIN documents d ON d.id = c.document_id
		JOIN document_versions dv ON dv.id = c.document_version_id
		WHERE c.collection_id = $2 AND d.status = 'active' AND dv.status = 'ready' AND d.latest_version = dv.version
		  AND to_tsvector('simple', c.content_lexical) @@ plainto_tsquery('simple', $1)
		ORDER BY lexical_score DESC
		LIMIT $3`,
		normalizedQuery, collectionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]RetrievalHit, 0)
	for rows.Next() {
		var (
			hit         RetrievalHit
			metadataRaw string
		)
		if err = rows.Scan(&hit.ChunkID, &hit.DocumentID, &hit.DocumentVersionID, &hit.DocumentName, &hit.CitationLabel, &hit.Content, &metadataRaw, &hit.LexicalScore); err != nil {
			return nil, err
		}
		if metadataRaw != "" {
			_ = json.Unmarshal([]byte(metadataRaw), &hit.Metadata)
		}
		hits = append(hits, hit)
	}
	if len(hits) > 0 {
		return hits, rows.Err()
	}

	rows, err = s.db.QueryContext(ctx,
		`SELECT c.id, c.document_id, c.document_version_id, d.name, c.citation_label, c.content, c.metadata,
		        similarity(c.content, $1) AS lexical_score
		FROM chunks c
		JOIN documents d ON d.id = c.document_id
		JOIN document_versions dv ON dv.id = c.document_version_id
		WHERE c.collection_id = $2 AND d.status = 'active' AND dv.status = 'ready' AND d.latest_version = dv.version
		  AND c.content ILIKE '%' || $1 || '%'
		ORDER BY lexical_score DESC
		LIMIT $3`,
		rawQuery, collectionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits = make([]RetrievalHit, 0)
	for rows.Next() {
		var (
			hit         RetrievalHit
			metadataRaw string
		)
		if err = rows.Scan(&hit.ChunkID, &hit.DocumentID, &hit.DocumentVersionID, &hit.DocumentName, &hit.CitationLabel, &hit.Content, &metadataRaw, &hit.LexicalScore); err != nil {
			return nil, err
		}
		if metadataRaw != "" {
			_ = json.Unmarshal([]byte(metadataRaw), &hit.Metadata)
		}
		hits = append(hits, hit)
	}
	return hits, rows.Err()
}

func loadDocumentsFromBytes(ctx context.Context, name, fileMD5 string, data []byte) ([]schema.Document, error) {
	tmpFile, err := os.CreateTemp("", "tinyclaw-knowledge-*"+filepath.Ext(name))
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err = tmpFile.Write(data); err != nil {
		return nil, err
	}
	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var loader documentloaders.Loader
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt":
		loader = documentloaders.NewText(tmpFile)
	case ".pdf":
		stat, statErr := tmpFile.Stat()
		if statErr != nil {
			return nil, statErr
		}
		loader = documentloaders.NewPDF(tmpFile, stat.Size())
	case ".csv":
		loader = documentloaders.NewCSV(tmpFile)
	case ".html":
		loader = documentloaders.NewHTML(tmpFile)
	default:
		return nil, fmt.Errorf("unsupported document type: %s", name)
	}

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(conf.KnowledgeConfInfo.ChunkSize),
		textsplitter.WithChunkOverlap(conf.KnowledgeConfInfo.ChunkOverlap),
		textsplitter.WithSeparators(conf.DefaultSpliter),
	)

	docs, err := loader.LoadAndSplit(ctx, splitter)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		if docs[i].Metadata == nil {
			docs[i].Metadata = map[string]interface{}{}
		}
		docs[i].Metadata["file_name"] = name
		docs[i].Metadata["file_md5"] = fileMD5
	}
	return docs, nil
}

func normalizeLexicalText(text string) string {
	var builder strings.Builder
	for _, r := range strings.TrimSpace(text) {
		switch {
		case unicode.Is(unicode.Han, r):
			builder.WriteRune(r)
			builder.WriteByte(' ')
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
		default:
			builder.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func fuseRetrievalHits(denseHits, lexicalHits []RetrievalHit) []RetrievalHit {
	merged := make(map[int64]*RetrievalHit)

	for idx, hit := range denseHits {
		existing := merged[hit.ChunkID]
		if existing == nil {
			copyHit := hit
			existing = &copyHit
			merged[hit.ChunkID] = existing
		}
		existing.DenseScore = hit.DenseScore
		existing.RRFScore += 1 / (rrfK + float64(idx+1))
	}

	for idx, hit := range lexicalHits {
		existing := merged[hit.ChunkID]
		if existing == nil {
			copyHit := hit
			existing = &copyHit
			merged[hit.ChunkID] = existing
		}
		existing.LexicalScore = hit.LexicalScore
		existing.RRFScore += 1 / (rrfK + float64(idx+1))
	}

	result := make([]RetrievalHit, 0, len(merged))
	for _, hit := range merged {
		hit.FinalScore = hit.RRFScore
		result = append(result, *hit)
	}

	sort.Slice(result, func(i, j int) bool {
		if math.Abs(result[i].FinalScore-result[j].FinalScore) > 1e-9 {
			return result[i].FinalScore > result[j].FinalScore
		}
		if math.Abs(result[i].DenseScore-result[j].DenseScore) > 1e-9 {
			return result[i].DenseScore > result[j].DenseScore
		}
		return result[i].LexicalScore > result[j].LexicalScore
	})

	if len(result) > 8 {
		result = result[:8]
	}
	return result
}

type retrievalThresholds struct {
	DenseMin    float64
	LexicalMin  float64
	FusedMin    float64
	RerankerMin float64
}

func currentRetrievalThresholds() retrievalThresholds {
	return retrievalThresholds{
		DenseMin:    conf.KnowledgeConfInfo.DenseScoreThreshold,
		LexicalMin:  conf.KnowledgeConfInfo.LexicalScoreThreshold,
		FusedMin:    conf.KnowledgeConfInfo.FusedScoreThreshold,
		RerankerMin: conf.KnowledgeConfInfo.RerankerScoreThreshold,
	}
}

func filterRetrievalHits(hits []RetrievalHit, thresholds retrievalThresholds) []RetrievalHit {
	if len(hits) == 0 {
		return nil
	}

	filtered := make([]RetrievalHit, 0, len(hits))
	for _, hit := range hits {
		if !isRelevantRetrievalHit(hit, thresholds) {
			continue
		}
		filtered = append(filtered, hit)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if math.Abs(filtered[i].FinalScore-filtered[j].FinalScore) > 1e-9 {
			return filtered[i].FinalScore > filtered[j].FinalScore
		}
		if math.Abs(filtered[i].RerankScore-filtered[j].RerankScore) > 1e-9 {
			return filtered[i].RerankScore > filtered[j].RerankScore
		}
		if math.Abs(filtered[i].RRFScore-filtered[j].RRFScore) > 1e-9 {
			return filtered[i].RRFScore > filtered[j].RRFScore
		}
		if math.Abs(filtered[i].DenseScore-filtered[j].DenseScore) > 1e-9 {
			return filtered[i].DenseScore > filtered[j].DenseScore
		}
		return filtered[i].LexicalScore > filtered[j].LexicalScore
	})

	if len(filtered) > 8 {
		filtered = filtered[:8]
	}
	return filtered
}

func isRelevantRetrievalHit(hit RetrievalHit, thresholds retrievalThresholds) bool {
	if hit.Reranked {
		return hit.RerankScore >= thresholds.RerankerMin
	}
	if hit.DenseScore >= thresholds.DenseMin {
		return true
	}
	if hit.LexicalScore >= thresholds.LexicalMin {
		return true
	}
	return hit.FinalScore >= thresholds.FusedMin
}

func hitCitations(hits []RetrievalHit) []string {
	citations := make([]string, 0, len(hits))
	for _, hit := range hits {
		if hit.CitationLabel == "" {
			continue
		}
		citations = append(citations, "- "+hit.CitationLabel)
	}
	return citations
}

func buildKnowledgePrompt(userInput string, hits []RetrievalHit) string {
	if len(hits) == 0 {
		return fmt.Sprintf("User question:\n%s\n\nNo relevant knowledge snippets were retrieved. Answer cautiously and say when the knowledge base does not contain enough evidence.", userInput)
	}

	var builder strings.Builder
	builder.WriteString("You are answering with retrieved knowledge snippets.\n")
	builder.WriteString("Rules:\n")
	builder.WriteString("1. Ground the answer in the snippets.\n")
	builder.WriteString("2. Cite snippets inline using their labels like [D1].\n")
	builder.WriteString("3. If the snippets are insufficient, clearly say so.\n\n")
	builder.WriteString("User question:\n")
	builder.WriteString(userInput)
	builder.WriteString("\n\nKnowledge snippets:\n")
	for _, hit := range hits {
		builder.WriteString(hit.CitationLabel)
		builder.WriteString("\n")
		builder.WriteString(hit.Content)
		builder.WriteString("\n\n")
	}
	return builder.String()
}

func noRelevantKnowledgeAnswer() string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(conf.BaseConfInfo.Lang)), "zh") {
		return "知识库里没有足够相关的内容来回答这个问题。请换一种问法，或先补充相关知识文档。"
	}
	return "The knowledge base does not contain enough relevant information to answer this question. Please rephrase it or add supporting documents first."
}

func containsCitation(answer string) bool {
	return strings.Contains(answer, "[D")
}

func checksumMD5(data []byte) string {
	sum := md5.Sum(data)
	return hex.EncodeToString(sum[:])
}

func vectorLiteral(values []float32) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%f", value))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func contentTypeFromName(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".html":
		return "text/html"
	case ".pdf":
		return "application/pdf"
	default:
		return "application/octet-stream"
	}
}

func mustJSON(value interface{}) string {
	body, err := json.Marshal(value)
	if err != nil {
		return "[]"
	}
	return string(body)
}

func answerWithKnowledge(ctx context.Context, callLLM *llm.LLM, prompt string, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}

	answer, _, err := defaultService.answerWithLLM(ctx, callLLM, prompt)
	if err != nil {
		return nil, err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: answer,
			},
		},
	}, nil
}
