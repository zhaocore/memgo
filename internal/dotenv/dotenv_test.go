package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

// 覆盖: 注释/export/引号/# 值/行内注释/不覆盖已导出变量/缺文件跳过/语法错误报错。
func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, ".env")
	src := "# 注释\n\nexport PLAIN=abc\nQUOTED=\"hello world\"\nHASH=# 戳\nINLINE=val # 行内注释\n"
	if err := os.WriteFile(good, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EXISTING", "keep")

	if err := LoadFile(good); err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	for k, want := range map[string]string{
		"PLAIN": "abc", "QUOTED": "hello world", "HASH": "# 戳", "INLINE": "val", "EXISTING": "keep",
	} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s = %q, want %q", k, got, want)
		}
	}

	if err := LoadFile(filepath.Join(dir, "nope.env")); err != nil {
		t.Errorf("缺文件应跳过, got %v", err)
	}
	bad := filepath.Join(dir, "bad.env")
	_ = os.WriteFile(bad, []byte("GOOD=x\nBROKEN\n"), 0o600)
	if err := LoadFile(bad); err == nil {
		t.Error("语法错误应报错, got nil")
	}
}
