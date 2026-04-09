package http

import (
	"encoding/base64"
	"net/http"

	"github.com/LittleSongxx/TinyClaw/knowledge"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/utils"
)

type KnowledgeFile struct {
	FileName string `json:"file_name"`
	Content  string `json:"content"`
}

type KnowledgeCollectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type KnowledgeDocumentRequest struct {
	FileName     string `json:"file_name"`
	Content      string `json:"content"`
	SourceType   string `json:"source_type"`
	ContentType  string `json:"content_type"`
	DataBase64   string `json:"data_base64"`
	Collection   string `json:"collection_name"`
	DocumentName string `json:"document_name"`
}

type KnowledgeRetrievalRequest struct {
	Query string `json:"query"`
}

func CreateKnowledgeFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	knowledgeFile := &KnowledgeFile{}
	err := utils.HandleJsonBody(r, knowledgeFile)
	if err != nil {
		logger.ErrorCtx(ctx, "parse json body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	document, version, job, err := knowledge.CreateTextDocument(ctx, knowledgeFile.FileName, knowledgeFile.Content)
	if err != nil {
		logger.ErrorCtx(ctx, "create knowledge document fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, map[string]interface{}{
		"document": document,
		"version":  version,
		"job":      job,
	})
}

func GetKnowledgeFileContent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := r.ParseForm()
	if err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	name := r.FormValue("file_name")
	if name == "" {
		name = r.FormValue("name")
	}

	content, err := knowledge.GetDocumentContent(ctx, name)
	if err != nil {
		logger.ErrorCtx(ctx, "get knowledge document content fail", "err", err, "name", name)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, map[string]interface{}{"content": content})
}

func DeleteKnowledgeFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := r.ParseForm()
	if err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	fileName := r.FormValue("file_name")
	if fileName == "" {
		fileName = r.FormValue("name")
	}

	err = knowledge.DeleteDocumentByName(ctx, fileName)
	if err != nil {
		logger.ErrorCtx(ctx, "delete knowledge document fail", "err", err, "file_name", fileName)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, nil)
}

func ListKnowledgeFiles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := r.ParseForm()
	if err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	page := utils.ParseInt(r.FormValue("page"))
	pageSize := utils.ParseInt(r.FormValue("page_size"))
	if pageSize == 0 {
		pageSize = utils.ParseInt(r.FormValue("pageSize"))
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	name := r.FormValue("name")

	documents, err := knowledge.ListDocuments(ctx, page, pageSize, name)
	if err != nil {
		logger.ErrorCtx(ctx, "list knowledge documents fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	list := make([]map[string]interface{}, 0, len(documents.List))
	for _, document := range documents.List {
		list = append(list, map[string]interface{}{
			"id":             document.ID,
			"file_name":      document.Name,
			"name":           document.Name,
			"source_type":    document.SourceType,
			"content_type":   document.ContentType,
			"status":         document.Status,
			"current_status": document.CurrentStatus,
			"chunk_count":    document.ChunkCount,
			"latest_version": document.LatestVersion,
			"create_time":    document.CreateTime,
			"update_time":    document.UpdateTime,
		})
	}

	utils.Success(ctx, w, r, map[string]interface{}{
		"list":      list,
		"total":     documents.Total,
		"page":      documents.Page,
		"page_size": documents.PageSize,
	})
}

func ClearKnowledgeData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	for {
		documents, err := knowledge.ListDocuments(ctx, 1, 100, "")
		if err != nil {
			logger.ErrorCtx(ctx, "list documents fail", "err", err)
			utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
			return
		}

		if len(documents.List) == 0 {
			break
		}

		for _, document := range documents.List {
			if err := knowledge.DeleteDocumentByName(ctx, document.Name); err != nil {
				logger.ErrorCtx(ctx, "delete document fail", "err", err, "name", document.Name)
				utils.Failure(ctx, w, r, param.CodeDBWriteFail, param.MsgDBWriteFail, err)
				return
			}
		}
	}

	utils.Success(ctx, w, r, nil)
}

func ListKnowledgeCollections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := knowledge.ListCollections(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "list knowledge collections fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func CreateKnowledgeCollection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &KnowledgeCollectionRequest{}
	if err := utils.HandleJsonBody(r, req); err != nil {
		logger.ErrorCtx(ctx, "parse json body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	collection, err := knowledge.CreateCollection(ctx, req.Name, req.Description)
	if err != nil {
		logger.ErrorCtx(ctx, "create knowledge collection fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, collection)
}

func ListKnowledgeDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	page := utils.ParseInt(r.FormValue("page"))
	pageSize := utils.ParseInt(r.FormValue("page_size"))
	if pageSize == 0 {
		pageSize = utils.ParseInt(r.FormValue("pageSize"))
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	result, err := knowledge.ListDocuments(ctx, page, pageSize, r.FormValue("name"))
	if err != nil {
		logger.ErrorCtx(ctx, "list knowledge documents fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func GetKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	name := r.FormValue("file_name")
	if name == "" {
		name = r.FormValue("name")
	}

	content, err := knowledge.GetDocumentContent(ctx, name)
	if err != nil {
		logger.ErrorCtx(ctx, "get knowledge document content fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, map[string]interface{}{"content": content, "file_name": name})
}

func CreateKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &KnowledgeDocumentRequest{}
	if err := utils.HandleJsonBody(r, req); err != nil {
		logger.ErrorCtx(ctx, "parse json body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	if req.FileName == "" {
		req.FileName = req.DocumentName
	}

	if req.DataBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(req.DataBase64)
		if err != nil {
			logger.ErrorCtx(ctx, "decode document payload fail", "err", err)
			utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
			return
		}
		sourceType := req.SourceType
		if sourceType == "" {
			sourceType = "upload"
		}
		contentType := req.ContentType
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
		document, version, job, err := knowledge.CreateBinaryDocument(ctx, req.FileName, sourceType, contentType, data)
		if err != nil {
			logger.ErrorCtx(ctx, "create binary knowledge document fail", "err", err)
			utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
			return
		}
		utils.Success(ctx, w, r, map[string]interface{}{
			"document": document,
			"version":  version,
			"job":      job,
		})
		return
	}

	document, version, job, err := knowledge.CreateTextDocument(ctx, req.FileName, req.Content)
	if err != nil {
		logger.ErrorCtx(ctx, "create text knowledge document fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, map[string]interface{}{
		"document": document,
		"version":  version,
		"job":      job,
	})
}

func DeleteKnowledgeDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}
	name := r.FormValue("file_name")
	if name == "" {
		name = r.FormValue("name")
	}
	if err := knowledge.DeleteDocumentByName(ctx, name); err != nil {
		logger.ErrorCtx(ctx, "delete knowledge document fail", "err", err, "name", name)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, nil)
}

func ListKnowledgeJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	page := utils.ParseInt(r.FormValue("page"))
	pageSize := utils.ParseInt(r.FormValue("page_size"))
	if pageSize == 0 {
		pageSize = utils.ParseInt(r.FormValue("pageSize"))
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	result, err := knowledge.ListIngestionJobs(ctx, page, pageSize, r.FormValue("status"))
	if err != nil {
		logger.ErrorCtx(ctx, "list knowledge jobs fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func DebugKnowledgeRetrieval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &KnowledgeRetrievalRequest{}
	if err := utils.HandleJsonBody(r, req); err != nil {
		logger.ErrorCtx(ctx, "parse json body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	result, err := knowledge.DebugRetrieve(ctx, req.Query)
	if err != nil {
		logger.ErrorCtx(ctx, "debug knowledge retrieval fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func ListKnowledgeRetrievalRuns(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	page := utils.ParseInt(r.FormValue("page"))
	pageSize := utils.ParseInt(r.FormValue("page_size"))
	if pageSize == 0 {
		pageSize = utils.ParseInt(r.FormValue("pageSize"))
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	result, err := knowledge.ListRetrievalRuns(ctx, page, pageSize)
	if err != nil {
		logger.ErrorCtx(ctx, "list knowledge retrieval runs fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func GetKnowledgeRetrievalRun(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := r.ParseForm(); err != nil {
		logger.ErrorCtx(ctx, "parse form error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	id := utils.ParseInt(r.FormValue("id"))
	if id == 0 {
		id = utils.ParseInt(r.FormValue("run_id"))
	}
	if id == 0 {
		utils.Failure(ctx, w, r, param.CodeParamError, "ID is required", nil)
		return
	}

	result, err := knowledge.GetRetrievalRun(ctx, int64(id))
	if err != nil {
		logger.ErrorCtx(ctx, "get knowledge retrieval run fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}
