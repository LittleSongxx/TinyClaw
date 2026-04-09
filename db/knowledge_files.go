package db

import (
	"fmt"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
)

const knowledgeFileTable = "knowledge_files"

type KnowledgeFileRecord struct {
	ID         int64  `json:"id"`
	FileName   string `json:"file_name"`
	FileMd5    string `json:"file_md5"`
	VectorId   string `json:"vector_id"`
	UpdateTime int64  `json:"update_time"`
	CreateTime int    `json:"create_time"`
	IsDeleted  int    `json:"is_deleted"`
}

func InsertKnowledgeFile(fileName, fileMd5 string) (int64, error) {
	insertSQL := fmt.Sprintf(`INSERT INTO %s (file_name, file_md5, create_time, update_time, vector_id, from_bot) VALUES (?, ?, ?, ?, ?, ?)`, knowledgeFileTable)
	result, err := DB.Exec(insertSQL, fileName, fileMd5, time.Now().Unix(), time.Now().Unix(), "", conf.BaseConfInfo.BotName)
	if err != nil {
		return 0, err
	}

	// get last insert id
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetKnowledgeFileByFileMD5(fileMd5 string) ([]*KnowledgeFileRecord, error) {
	querySQL := fmt.Sprintf(`SELECT id, file_name, file_md5, update_time, create_time FROM %s WHERE file_md5 = ? and is_deleted = 0 and from_bot = ?`, knowledgeFileTable)
	rows, err := DB.Query(querySQL, fileMd5, conf.BaseConfInfo.BotName)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*KnowledgeFileRecord
	for rows.Next() {
		var record KnowledgeFileRecord
		if err := rows.Scan(&record.ID, &record.FileName, &record.FileMd5, &record.UpdateTime, &record.CreateTime); err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func GetKnowledgeFileByFileName(fileName string) ([]*KnowledgeFileRecord, error) {
	querySQL := fmt.Sprintf(`SELECT id, file_name, file_md5, update_time, create_time, vector_id FROM %s WHERE file_name = ? and is_deleted = 0 and from_bot = ?`, knowledgeFileTable)
	rows, err := DB.Query(querySQL, fileName, conf.BaseConfInfo.BotName)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*KnowledgeFileRecord
	for rows.Next() {
		var record KnowledgeFileRecord
		if err := rows.Scan(&record.ID, &record.FileName, &record.FileMd5, &record.UpdateTime, &record.CreateTime, &record.VectorId); err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func DeleteKnowledgeFileByFileName(fileName string) error {
	query := fmt.Sprintf(`UPDATE %s set is_deleted = 1 WHERE file_name = ? and from_bot = ?`, knowledgeFileTable)
	_, err := DB.Exec(query, fileName, conf.BaseConfInfo.BotName)
	return err
}

func DeleteAllKnowledgeFiles() error {
	query := fmt.Sprintf(`UPDATE %s set is_deleted = 1 WHERE is_deleted = 0 and from_bot = ?`, knowledgeFileTable)
	_, err := DB.Exec(query, conf.BaseConfInfo.BotName)
	return err
}

func DeleteKnowledgeFileByVectorID(vectorID string) error {
	query := fmt.Sprintf(`UPDATE %s set is_deleted = 1 WHERE vector_id = ? and from_bot = ?`, knowledgeFileTable)
	_, err := DB.Exec(query, vectorID, conf.BaseConfInfo.BotName)
	return err
}

func UpdateVectorIdByFileMd5(fileMd5, vectorId string) error {
	query := fmt.Sprintf(`UPDATE %s set vector_id = ? WHERE file_md5 = ? and from_bot = ?`, knowledgeFileTable)
	_, err := DB.Exec(query, vectorId, fileMd5, conf.BaseConfInfo.BotName)
	return err
}

func GetKnowledgeFilesByPage(page, pageSize int, name string) ([]KnowledgeFileRecord, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	var (
		whereSQL = "WHERE is_deleted = 0 and from_bot = ?"
		args     = []interface{}{conf.BaseConfInfo.BotName}
	)

	if name != "" {
		whereSQL += " AND file_name LIKE ?"
		args = append(args, "%"+name+"%")
	}

	// 查询数据
	listSQL := fmt.Sprintf(`
		SELECT id, file_name, file_md5, vector_id, update_time, create_time, is_deleted
		FROM %s %s
		ORDER BY id DESC
		LIMIT ? OFFSET ?`, knowledgeFileTable, whereSQL)

	args = append(args, pageSize, offset)

	rows, err := DB.Query(listSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []KnowledgeFileRecord
	for rows.Next() {
		var f KnowledgeFileRecord
		if err := rows.Scan(&f.ID, &f.FileName, &f.FileMd5, &f.VectorId, &f.UpdateTime, &f.CreateTime, &f.IsDeleted); err != nil {
			return nil, err
		}
		files = append(files, f)
	}

	return files, nil
}

func GetKnowledgeFilesCount(name string) (int, error) {
	whereSQL := "WHERE is_deleted = 0 and from_bot = ?"
	args := []interface{}{conf.BaseConfInfo.BotName}

	if name != "" {
		whereSQL += " AND file_name LIKE ?"
		args = append(args, "%"+name+"%")
	}

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s %s", knowledgeFileTable, whereSQL)

	var count int
	err := DB.QueryRow(countSQL, args...).Scan(&count)

	return count, err
}
