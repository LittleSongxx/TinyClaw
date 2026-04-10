package db

import (
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestInsertAndGetKnowledgeFile(t *testing.T) {

	suffix := time.Now().Format("150405.000000000")
	fileName := "test-" + suffix + ".txt"
	fileMd5 := "abc123-" + suffix

	// 插入
	id, err := InsertKnowledgeFile(fileName, fileMd5)
	assert.NoError(t, err)
	assert.NotZero(t, id)

	// 按 md5 查询
	files, err := GetKnowledgeFileByFileMD5(fileMd5)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, fileName, files[0].FileName)

	// 按 fileName 查询
	files2, err := GetKnowledgeFileByFileName(fileName)
	assert.NoError(t, err)
	assert.Len(t, files2, 1)
	assert.Equal(t, fileMd5, files2[0].FileMd5)
}

func TestUpdateAndDeleteKnowledgeFile(t *testing.T) {
	suffix := time.Now().Format("150405.000000000")
	fileName := "test-" + suffix + ".txt"
	fileMd5 := "abc123-" + suffix

	// 插入
	_, err := InsertKnowledgeFile(fileName, fileMd5)
	assert.NoError(t, err)

	// 更新 vector_id
	newVector := "vec-1"
	err = UpdateVectorIdByFileMd5(fileMd5, newVector)
	assert.NoError(t, err)

	files, err := GetKnowledgeFileByFileName(fileName)
	assert.NoError(t, err)
	assert.Equal(t, newVector, files[0].VectorId)

	// 删除 by fileName
	err = DeleteKnowledgeFileByFileName(fileName)
	assert.NoError(t, err)

	files, err = GetKnowledgeFileByFileName(fileName)
	assert.NoError(t, err)
	assert.Len(t, files, 0)

	// 再插入一个
	secondName := "b-" + suffix + ".txt"
	secondMd5 := "def456-" + suffix
	_, err = InsertKnowledgeFile(secondName, secondMd5)
	assert.NoError(t, err)

	// 删除 by vectorId
	err = UpdateVectorIdByFileMd5(secondMd5, "vec-2-"+suffix)
	assert.NoError(t, err)

	err = DeleteKnowledgeFileByVectorID("vec-2-" + suffix)
	assert.NoError(t, err)

	files, err = GetKnowledgeFileByFileName(secondName)
	assert.NoError(t, err)
	assert.Len(t, files, 0)
}

func TestInsertTimeStamps(t *testing.T) {
	suffix := time.Now().Format("150405.000000000")
	fileName := "time-" + suffix + ".txt"
	fileMd5 := "time123-" + suffix

	// 插入
	_, err := InsertKnowledgeFile(fileName, fileMd5)
	assert.NoError(t, err)

	files, err := GetKnowledgeFileByFileMD5(fileMd5)
	assert.NoError(t, err)
	assert.Len(t, files, 1)

	now := time.Now().Unix()
	assert.LessOrEqual(t, files[0].UpdateTime, now)
}
