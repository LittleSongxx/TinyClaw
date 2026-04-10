package utils

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/LittleSongxx/TinyClaw/logger"
)

func GetAbsPath(relPath string) string {
	baseDir := resolveBaseDir()
	if baseDir == "" {
		logger.Error("Failed to resolve base directory")
		return ""
	}
	return filepath.Join(baseDir, relPath)
}

func GetTailStartOffset(filePath string, lines int) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	const bufferSize = 4096
	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	size := stat.Size()
	var offset = size
	var count int

	for offset > 0 && count <= lines {
		readSize := int64(bufferSize)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize
		tmp := make([]byte, readSize)
		if _, err := file.ReadAt(tmp, offset); err != nil {
			return 0, err
		}
		count += bytes.Count(tmp, []byte("\n"))
	}

	if offset <= 0 {
		offset = 0
	}

	return offset, nil
}

func resolveBaseDir() string {
	if root := strings.TrimSpace(os.Getenv("TINYCLAW_ROOT")); root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
		return root
	}

	candidates := make([]string, 0, 2)
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Dir(exe))
	}

	for _, candidate := range candidates {
		if root := findProjectRoot(candidate); root != "" {
			return root
		}
	}

	return ""
}

func findProjectRoot(start string) string {
	current := filepath.Clean(start)
	for current != "." && current != string(filepath.Separator) {
		if looksLikeProjectRoot(current) {
			return current
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	if looksLikeProjectRoot(current) {
		return current
	}
	return ""
}

func looksLikeProjectRoot(dir string) bool {
	if dir == "" {
		return false
	}

	if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
		return true
	}

	confInfo, confErr := os.Stat(filepath.Join(dir, "conf"))
	cmdInfo, cmdErr := os.Stat(filepath.Join(dir, "cmd"))
	return confErr == nil && confInfo.IsDir() && cmdErr == nil && cmdInfo.IsDir()
}
