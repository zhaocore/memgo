// Package dotenv: 轻量 .env 加载器 (godotenv 子集语义, 零依赖)。
// 支持: KEY=VALUE、# 注释、export 前缀、单双引号包裹、行内注释 (# 前有空格)。
// 语义: 已导出的环境变量不覆盖 (真实环境优先); 文件不存在视为无 .env, 不报错;
//
//	语法错误报错并含文件名与行号 (fail loud)。
//
// ponytail: 不处理双引号内转义 (\n) 与多行值 —— 值含特殊字符时用引号包裹即可;
//
//	需要完整 godotenv 语义时换 github.com/joho/godotenv。
package dotenv

import (
	"fmt"
	"os"
	"strings"
)

// Load 读入当前工作目录下的 .env。
func Load() error {
	return LoadFile(".env")
}

// LoadFile 逐行读入指定路径的 .env 并写入进程环境。
func LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("dotenv: 读取 %s 失败: %w", path, err)
	}
	for i, line := range strings.Split(string(data), "\n") {
		key, val, err := parseLine(line)
		if err != nil {
			return fmt.Errorf("dotenv: %s 第 %d 行: %w", path, i+1, err)
		}
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, val); err != nil {
				return fmt.Errorf("dotenv: 设置 %s 失败: %w", key, err)
			}
		}
	}
	return nil
}

// parseLine 解析单行; key 为空表示空行或注释。
func parseLine(line string) (string, string, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", nil
	}
	line = strings.TrimPrefix(line, "export ")
	key, rest, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", fmt.Errorf("缺少 '='")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", fmt.Errorf("key 为空")
	}
	val := strings.TrimSpace(rest)
	if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') {
		quote := val[0]
		if val[len(val)-1] != quote {
			return "", "", fmt.Errorf("引号未闭合: %s", val)
		}
		val = val[1 : len(val)-1]
	} else if i := strings.Index(val, " #"); i >= 0 {
		val = strings.TrimSpace(val[:i])
	}
	return key, val, nil
}
