package knowledge

import (
	"context"
	"fmt"
)

func ListCollections(ctx context.Context) (*ListResult[Collection], error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.listCollections(ctx)
}

func CreateCollection(ctx context.Context, name, description string) (*Collection, error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	kb, _, err := defaultService.ensureDefaultCollection(ctx)
	if err != nil {
		return nil, err
	}
	return defaultService.getOrCreateCollection(ctx, kb.ID, name, description)
}

func CreateTextDocument(ctx context.Context, name, content string) (*Document, *DocumentVersion, *IngestionJob, error) {
	if defaultService == nil {
		return nil, nil, nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.upsertDocumentContent(ctx, "", name, "text", contentTypeFromName(name), []byte(content))
}

func CreateBinaryDocument(ctx context.Context, name, sourceType, contentType string, data []byte) (*Document, *DocumentVersion, *IngestionJob, error) {
	if defaultService == nil {
		return nil, nil, nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.upsertDocumentContent(ctx, "", name, sourceType, contentType, data)
}

func ListDocuments(ctx context.Context, page, pageSize int, name string) (*ListResult[Document], error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.listDocuments(ctx, page, pageSize, name)
}

func ListIngestionJobs(ctx context.Context, page, pageSize int, status string) (*ListResult[IngestionJob], error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.listIngestionJobs(ctx, page, pageSize, status)
}

func GetDocumentContent(ctx context.Context, name string) (string, error) {
	if defaultService == nil {
		return "", fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.getDocumentContent(ctx, name)
}

func DeleteDocumentByName(ctx context.Context, name string) error {
	if defaultService == nil {
		return fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.deleteDocumentByName(ctx, name)
}

func DebugRetrieve(ctx context.Context, query string) (*RetrievalDebugResult, error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.debugRetrieve(ctx, query, true)
}

func ListRetrievalRuns(ctx context.Context, page, pageSize int) (*ListResult[RetrievalRun], error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.listRetrievalRuns(ctx, page, pageSize)
}

func GetRetrievalRun(ctx context.Context, id int64) (*RetrievalDebugResult, error) {
	if defaultService == nil {
		return nil, fmt.Errorf("knowledge service is not enabled")
	}
	return defaultService.getRetrievalRun(ctx, id)
}
