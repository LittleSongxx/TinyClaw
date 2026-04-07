package rag

import (
	"context"
	"fmt"
)

func ListCollections(ctx context.Context) (*ListResult[Collection], error) {
	if defaultKnowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.listCollections(ctx)
}

func CreateCollection(ctx context.Context, name, description string) (*Collection, error) {
	if defaultKnowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	kb, _, err := defaultKnowledgeService.ensureDefaultCollection(ctx)
	if err != nil {
		return nil, err
	}
	return defaultKnowledgeService.getOrCreateCollection(ctx, kb.ID, name, description)
}

func CreateTextDocument(ctx context.Context, name, content string) (*Document, *DocumentVersion, *IngestionJob, error) {
	if defaultKnowledgeService == nil {
		return nil, nil, nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.upsertDocumentContent(ctx, "", name, "text", contentTypeFromName(name), []byte(content))
}

func CreateBinaryDocument(ctx context.Context, name, sourceType, contentType string, data []byte) (*Document, *DocumentVersion, *IngestionJob, error) {
	if defaultKnowledgeService == nil {
		return nil, nil, nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.upsertDocumentContent(ctx, "", name, sourceType, contentType, data)
}

func ListDocuments(ctx context.Context, page, pageSize int, name string) (*ListResult[Document], error) {
	if defaultKnowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.listDocuments(ctx, page, pageSize, name)
}

func ListIngestionJobs(ctx context.Context, page, pageSize int, status string) (*ListResult[IngestionJob], error) {
	if defaultKnowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.listIngestionJobs(ctx, page, pageSize, status)
}

func GetDocumentContent(ctx context.Context, name string) (string, error) {
	if defaultKnowledgeService == nil {
		return "", fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.getDocumentContent(ctx, name)
}

func DeleteDocumentByName(ctx context.Context, name string) error {
	if defaultKnowledgeService == nil {
		return fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.deleteDocumentByName(ctx, name)
}

func DebugRetrieve(ctx context.Context, query string) (*RetrievalDebugResult, error) {
	if defaultKnowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.debugRetrieve(ctx, query, true)
}

func ListRetrievalRuns(ctx context.Context, page, pageSize int) (*ListResult[RetrievalRun], error) {
	if defaultKnowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.listRetrievalRuns(ctx, page, pageSize)
}

func GetRetrievalRun(ctx context.Context, id int64) (*RetrievalDebugResult, error) {
	if defaultKnowledgeService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultKnowledgeService.getRetrievalRun(ctx, id)
}
