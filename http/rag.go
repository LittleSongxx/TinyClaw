package http

import (
	"encoding/base64"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/db"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/TinyClaw/param"
	"github.com/LittleSongxx/TinyClaw/rag"
	"github.com/LittleSongxx/TinyClaw/utils"
)

type RagFile struct {
	FileName string `json:"file_name"`
	Content  string `json:"content"`
}

type RagCollectionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RagDocumentRequest struct {
	FileName     string `json:"file_name"`
	Content      string `json:"content"`
	SourceType   string `json:"source_type"`
	ContentType  string `json:"content_type"`
	DataBase64   string `json:"data_base64"`
	Collection   string `json:"collection_name"`
	DocumentName string `json:"document_name"`
}

type RagRetrievalRequest struct {
	Query string `json:"query"`
}

func CreateRagFile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ragFile := &RagFile{}
	err := utils.HandleJsonBody(r, ragFile)
	if err != nil {
		logger.ErrorCtx(ctx, "parse json body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	if rag.KnowledgeV2Enabled() {
		document, version, job, err := rag.CreateTextDocument(ctx, ragFile.FileName, ragFile.Content)
		if err != nil {
			logger.ErrorCtx(ctx, "create rag document fail", "err", err)
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

	path := conf.RagConfInfo.KnowledgePath + "/" + ragFile.FileName
	_, err = os.Stat(path)
	fileNotExist := os.IsNotExist(err)

	err = os.WriteFile(path, []byte(ragFile.Content), 0644)
	if err != nil {
		logger.ErrorCtx(ctx, "write file error", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	if fileNotExist && conf.RagConfInfo.Store == nil {
		_, err = db.InsertRagFile(ragFile.FileName, "")
		if err != nil {
			logger.ErrorCtx(ctx, "insert rag file fail", "err", err)
			utils.Failure(ctx, w, r, param.CodeDBWriteFail, param.MsgDBWriteFail, err)
			return
		}
	}

	utils.Success(ctx, w, r, nil)
}

func GetRagFileContent(w http.ResponseWriter, r *http.Request) {
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

	if rag.KnowledgeV2Enabled() {
		content, err := rag.GetDocumentContent(ctx, name)
		if err != nil {
			logger.ErrorCtx(ctx, "get rag document content fail", "err", err, "name", name)
			utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
			return
		}
		utils.Success(ctx, w, r, map[string]interface{}{"content": content})
		return
	}

	if !strings.Contains(name, ".txt") {
		logger.ErrorCtx(ctx, "only support txt file")
		utils.Failure(ctx, w, r, param.CodeTxtFileOnly, param.MsgTxtFileOnly, "only support txt file")
		return
	}

	path := conf.RagConfInfo.KnowledgePath + "/" + name
	file, err := os.Open(path)
	if err != nil {
		logger.ErrorCtx(ctx, "open file error", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		logger.ErrorCtx(ctx, "read file error", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	utils.Success(ctx, w, r, map[string]interface{}{"content": string(content)})
}

func DeleteRagFile(w http.ResponseWriter, r *http.Request) {
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

	if rag.KnowledgeV2Enabled() {
		err = rag.DeleteDocumentByName(ctx, fileName)
		if err != nil {
			logger.ErrorCtx(ctx, "delete rag document fail", "err", err, "file_name", fileName)
			utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
			return
		}
		utils.Success(ctx, w, r, nil)
		return
	}

	err = os.Remove(conf.RagConfInfo.KnowledgePath + "/" + fileName)
	if err != nil {
		logger.ErrorCtx(ctx, "delete file error", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}

	if conf.RagConfInfo.Store == nil {
		err = db.DeleteRagFileByFileName(fileName)
		if err != nil {
			logger.ErrorCtx(ctx, "delete rag file fail", "err", err)
			utils.Failure(ctx, w, r, param.CodeDBWriteFail, param.MsgDBWriteFail, err)
			return
		}
	}

	utils.Success(ctx, w, r, nil)
}

func GetRagFile(w http.ResponseWriter, r *http.Request) {
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

	if rag.KnowledgeV2Enabled() {
		documents, err := rag.ListDocuments(ctx, page, pageSize, name)
		if err != nil {
			logger.ErrorCtx(ctx, "list rag documents fail", "err", err)
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
		return
	}

	ragFiles, err := db.GetRagFilesByPage(page, pageSize, name)
	if err != nil {
		logger.ErrorCtx(ctx, "get rag files error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}

	total, err := db.GetRagFilesCount(name)
	if err != nil {
		logger.ErrorCtx(ctx, "get rag file count error", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBWriteFail, err)
		return
	}

	utils.Success(ctx, w, r, map[string]interface{}{
		"list":  ragFiles,
		"total": total,
	})
}

func ClearAllVectorData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if rag.KnowledgeV2Enabled() {
		for {
			documents, err := rag.ListDocuments(ctx, 1, 100, "")
			if err != nil {
				logger.ErrorCtx(ctx, "list documents fail", "err", err)
				utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
				return
			}

			if len(documents.List) == 0 {
				break
			}

			for _, document := range documents.List {
				if err := rag.DeleteDocumentByName(ctx, document.Name); err != nil {
					logger.ErrorCtx(ctx, "delete document fail", "err", err, "name", document.Name)
					utils.Failure(ctx, w, r, param.CodeDBWriteFail, param.MsgDBWriteFail, err)
					return
				}
			}
		}

		utils.Success(ctx, w, r, nil)
		return
	}

	if conf.RagConfInfo.Store != nil {
		page := 1
		pageSize := 10
		for {
			ragFiles, err := db.GetRagFilesByPage(page, pageSize, "")
			if err != nil {
				logger.ErrorCtx(ctx, "get rag files error", "err", err)
				utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
				return
			}

			for _, ragFile := range ragFiles {
				err = db.DeleteRagFileByFileName(ragFile.FileName)
				if err != nil {
					logger.ErrorCtx(ctx, "delete rag file fail", "err", err)
					utils.Failure(ctx, w, r, param.CodeDBWriteFail, param.MsgDBWriteFail, err)
					return
				}

				err = rag.DeleteStoreData(ctx, ragFile.VectorId)
				if err != nil {
					logger.ErrorCtx(ctx, "delete vector store data fail", "err", err)
					utils.Failure(ctx, w, r, param.CodeDBWriteFail, param.MsgDBWriteFail, err)
					return
				}
			}

			if len(ragFiles) < 10 {
				break
			}
		}
	}

	utils.Success(ctx, w, r, nil)
}

func ListRagCollections(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	result, err := rag.ListCollections(ctx)
	if err != nil {
		logger.ErrorCtx(ctx, "list rag collections fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func CreateRagCollection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &RagCollectionRequest{}
	if err := utils.HandleJsonBody(r, req); err != nil {
		logger.ErrorCtx(ctx, "parse json body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	collection, err := rag.CreateCollection(ctx, req.Name, req.Description)
	if err != nil {
		logger.ErrorCtx(ctx, "create rag collection fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, collection)
}

func ListRagDocuments(w http.ResponseWriter, r *http.Request) {
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

	result, err := rag.ListDocuments(ctx, page, pageSize, r.FormValue("name"))
	if err != nil {
		logger.ErrorCtx(ctx, "list rag documents fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func GetRagDocument(w http.ResponseWriter, r *http.Request) {
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

	content, err := rag.GetDocumentContent(ctx, name)
	if err != nil {
		logger.ErrorCtx(ctx, "get rag document content fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, map[string]interface{}{"content": content, "file_name": name})
}

func CreateRagDocument(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &RagDocumentRequest{}
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
		document, version, job, err := rag.CreateBinaryDocument(ctx, req.FileName, sourceType, contentType, data)
		if err != nil {
			logger.ErrorCtx(ctx, "create binary rag document fail", "err", err)
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

	document, version, job, err := rag.CreateTextDocument(ctx, req.FileName, req.Content)
	if err != nil {
		logger.ErrorCtx(ctx, "create text rag document fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, map[string]interface{}{
		"document": document,
		"version":  version,
		"job":      job,
	})
}

func DeleteRagDocument(w http.ResponseWriter, r *http.Request) {
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
	if err := rag.DeleteDocumentByName(ctx, name); err != nil {
		logger.ErrorCtx(ctx, "delete rag document fail", "err", err, "name", name)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, nil)
}

func ListRagJobs(w http.ResponseWriter, r *http.Request) {
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

	result, err := rag.ListIngestionJobs(ctx, page, pageSize, r.FormValue("status"))
	if err != nil {
		logger.ErrorCtx(ctx, "list rag jobs fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func DebugRagRetrieval(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	req := &RagRetrievalRequest{}
	if err := utils.HandleJsonBody(r, req); err != nil {
		logger.ErrorCtx(ctx, "parse json body error", "err", err)
		utils.Failure(ctx, w, r, param.CodeParamError, param.MsgParamError, err)
		return
	}

	result, err := rag.DebugRetrieve(ctx, req.Query)
	if err != nil {
		logger.ErrorCtx(ctx, "debug rag retrieval fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeServerFail, param.MsgServerFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func ListRagRetrievalRuns(w http.ResponseWriter, r *http.Request) {
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

	result, err := rag.ListRetrievalRuns(ctx, page, pageSize)
	if err != nil {
		logger.ErrorCtx(ctx, "list rag retrieval runs fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}

func GetRagRetrievalRun(w http.ResponseWriter, r *http.Request) {
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

	result, err := rag.GetRetrievalRun(ctx, int64(id))
	if err != nil {
		logger.ErrorCtx(ctx, "get rag retrieval run fail", "err", err)
		utils.Failure(ctx, w, r, param.CodeDBQueryFail, param.MsgDBQueryFail, err)
		return
	}
	utils.Success(ctx, w, r, result)
}
