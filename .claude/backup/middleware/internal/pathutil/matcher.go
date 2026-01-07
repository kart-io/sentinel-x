// Package pathutil 提供路径匹配工具函数。
// 这是一个内部包，仅供 middleware 包使用。
package pathutil

import "strings"

// PathMatcher 是路径匹配函数类型。
type PathMatcher func(path string) bool

// NewPathMatcher 创建一个路径匹配器。
// 支持精确匹配（skipPaths）和前缀匹配（skipPrefixes）。
//
// 参数：
//   - skipPaths: 需要精确匹配的路径列表
//   - skipPrefixes: 需要前缀匹配的路径列表
//
// 返回：
//   - PathMatcher: 路径匹配函数，如果路径匹配则返回 true
//
// 示例：
//
//	matcher := NewPathMatcher(
//	    []string{"/health", "/metrics"},
//	    []string{"/api/v1/public"},
//	)
//	if matcher("/health") {
//	    // 跳过处理
//	}
func NewPathMatcher(skipPaths, skipPrefixes []string) PathMatcher {
	// 使用 map 优化精确匹配性能（O(1) vs O(n)）
	pathSet := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		pathSet[p] = true
	}

	return func(path string) bool {
		// 精确匹配
		if pathSet[path] {
			return true
		}

		// 前缀匹配
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		}

		return false
	}
}

// ShouldSkip 是一个便捷函数，直接检查路径是否应该跳过。
// 适用于不需要重复使用匹配器的场景。
//
// 参数：
//   - path: 要检查的路径
//   - skipPaths: 需要精确匹配的路径列表
//   - skipPrefixes: 需要前缀匹配的路径列表
//
// 返回：
//   - bool: 如果路径匹配则返回 true
//
// 示例：
//
//	if ShouldSkip(req.URL.Path, []string{"/health"}, []string{"/api/public"}) {
//	    // 跳过处理
//	}
func ShouldSkip(path string, skipPaths, skipPrefixes []string) bool {
	// 精确匹配
	for _, p := range skipPaths {
		if path == p {
			return true
		}
	}

	// 前缀匹配
	for _, prefix := range skipPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}
