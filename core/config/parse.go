package config

import (
	"os"
	"path/filepath"
)

// PGVectorDefaults 对齐 Python PGVectorConfig 字段默认值。
type PGVectorDefaults struct {
	DBName             string
	CollectionName     string
	EmbeddingModelDims int
	DiskANN            bool
	HNSW               bool
	MinConn            int
	MaxConn            int
}

// DefaultPGVector 返回 Python PGVectorConfig 默认值。
func DefaultPGVector() PGVectorDefaults {
	return PGVectorDefaults{
		DBName:             "postgres",
		CollectionName:     "memgo",
		EmbeddingModelDims: 1536,
		DiskANN:            false,
		HNSW:               true,
		MinConn:            1,
		MaxConn:            5,
	}
}

// DefaultHistoryDBPath 对齐 Python: $MEMGO_DIR 或 ~/.memgo/history.db。
func DefaultHistoryDBPath() string {
	if d := os.Getenv("MEMGO_DIR"); d != "" {
		return filepath.Join(d, "history.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// 无 home 环境退化为相对路径, 等价 Python expanduser 失败退化
		return "history.db"
	}
	return filepath.Join(home, ".memgo", "history.db")
}
