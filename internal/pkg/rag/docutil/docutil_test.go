package docutil_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kart-io/sentinel-x/internal/pkg/rag/docutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureDir(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "docutil_test_ensuredir")
	defer os.RemoveAll(tmpDir)

	// 创建目录
	err := docutil.EnsureDir(tmpDir)
	require.NoError(t, err)

	// 验证目录存在
	info, err := os.Stat(tmpDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())

	// 再次调用应该不会报错
	err = docutil.EnsureDir(tmpDir)
	assert.NoError(t, err)
}

func TestFindFiles(t *testing.T) {
	// 创建临时目录结构
	tmpDir := filepath.Join(os.TempDir(), "docutil_test_findfiles")
	defer os.RemoveAll(tmpDir)

	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0o755))

	// 创建测试文件
	testFiles := []string{
		filepath.Join(tmpDir, "file1.md"),
		filepath.Join(tmpDir, "file2.txt"),
		filepath.Join(tmpDir, "subdir", "file3.md"),
		filepath.Join(tmpDir, "subdir", "file4.mdx"),
	}

	for _, f := range testFiles {
		require.NoError(t, os.WriteFile(f, []byte("test"), 0o644))
	}

	// 查找 .md 文件
	mdFiles, err := docutil.FindFiles(tmpDir, []string{".md"})
	require.NoError(t, err)
	assert.Len(t, mdFiles, 2)

	// 查找 .md 和 .mdx 文件
	mdxFiles, err := docutil.FindFiles(tmpDir, []string{".md", ".mdx"})
	require.NoError(t, err)
	assert.Len(t, mdxFiles, 3)

	// 查找 .txt 文件
	txtFiles, err := docutil.FindFiles(tmpDir, []string{".txt"})
	require.NoError(t, err)
	assert.Len(t, txtFiles, 1)
}

func TestReadFileContent(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "docutil_test_readfile.txt")
	defer os.Remove(tmpFile)

	expectedContent := "Hello, World!\n这是测试内容。"
	require.NoError(t, os.WriteFile(tmpFile, []byte(expectedContent), 0o644))

	content, err := docutil.ReadFileContent(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, content)

	// 读取不存在的文件
	_, err = docutil.ReadFileContent("/nonexistent/file.txt")
	assert.Error(t, err)
}

func TestFileExists(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "docutil_test_exists.txt")
	defer os.Remove(tmpFile)

	// 文件不存在
	assert.False(t, docutil.FileExists(tmpFile))

	// 创建文件
	require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0o644))

	// 文件存在
	assert.True(t, docutil.FileExists(tmpFile))
}

func TestDirExists(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "docutil_test_direxists")
	defer os.RemoveAll(tmpDir)

	// 目录不存在
	assert.False(t, docutil.DirExists(tmpDir))

	// 创建目录
	require.NoError(t, os.MkdirAll(tmpDir, 0o755))

	// 目录存在
	assert.True(t, docutil.DirExists(tmpDir))

	// 文件不是目录
	tmpFile := filepath.Join(os.TempDir(), "docutil_test_notdir.txt")
	require.NoError(t, os.WriteFile(tmpFile, []byte("test"), 0o644))
	defer os.Remove(tmpFile)

	assert.False(t, docutil.DirExists(tmpFile))
}
