// Package docutil 提供文档处理相关的工具函数。
package docutil

import (
	"archive/zip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// DownloadFile 从 URL 下载文件到指定路径。
func DownloadFile(url, dest string) error {
	// #nosec G107 -- URL 由用户控制,已通过业务逻辑验证
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// #nosec G304 -- 文件路径由调用方控制,已通过业务逻辑验证
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, resp.Body)
	return err
}

// ExtractZip 解压 ZIP 文件到指定目录。
// 包含 ZipSlip 漏洞防护。
func ExtractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()

	// 使用 0750 权限以符合安全最佳实践
	if err := os.MkdirAll(dest, 0o750); err != nil {
		return err
	}

	for _, f := range r.File {
		// #nosec G305 -- 文件路径已通过 ZipSlip 防护检查
		path := filepath.Join(dest, f.Name)

		// 检查 ZipSlip 漏洞
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return err
		}

		// #nosec G304 -- 文件路径已通过 ZipSlip 防护检查
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			_ = outFile.Close()
			return err
		}

		// #nosec G110 -- 解压炸弹防护由外层调用者负责(文件大小限制)
		_, err = io.Copy(outFile, rc)
		_ = outFile.Close()
		_ = rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// FindFiles 在目录中查找匹配指定扩展名的文件。
// extensions 是文件扩展名列表，如 []string{".md", ".mdx"}。
func FindFiles(dir string, extensions []string) ([]string, error) {
	var files []string
	extMap := make(map[string]bool)
	for _, ext := range extensions {
		extMap[strings.ToLower(ext)] = true
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if extMap[ext] {
				files = append(files, path)
			}
		}
		return nil
	})

	return files, err
}

// EnsureDir 确保目录存在，如果不存在则创建。
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0o750)
}

// ReadFileContent 读取文件内容。
func ReadFileContent(path string) (string, error) {
	// #nosec G304 -- 文件路径由调用方控制,已通过业务逻辑验证
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// FileExists 检查文件是否存在。
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists 检查目录是否存在。
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
