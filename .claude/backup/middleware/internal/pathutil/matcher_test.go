package pathutil

import "testing"

func TestNewPathMatcher(t *testing.T) {
	tests := []struct {
		name         string
		skipPaths    []string
		skipPrefixes []string
		testPath     string
		want         bool
	}{
		{
			name:      "精确匹配 - 匹配成功",
			skipPaths: []string{"/health", "/metrics"},
			testPath:  "/health",
			want:      true,
		},
		{
			name:      "精确匹配 - 匹配失败",
			skipPaths: []string{"/health", "/metrics"},
			testPath:  "/api/users",
			want:      false,
		},
		{
			name:         "前缀匹配 - 匹配成功",
			skipPrefixes: []string{"/api/public", "/static"},
			testPath:     "/api/public/users",
			want:         true,
		},
		{
			name:         "前缀匹配 - 匹配失败",
			skipPrefixes: []string{"/api/public"},
			testPath:     "/api/private/users",
			want:         false,
		},
		{
			name:         "混合匹配 - 精确匹配成功",
			skipPaths:    []string{"/health"},
			skipPrefixes: []string{"/api/public"},
			testPath:     "/health",
			want:         true,
		},
		{
			name:         "混合匹配 - 前缀匹配成功",
			skipPaths:    []string{"/health"},
			skipPrefixes: []string{"/api/public"},
			testPath:     "/api/public/login",
			want:         true,
		},
		{
			name:         "混合匹配 - 都不匹配",
			skipPaths:    []string{"/health"},
			skipPrefixes: []string{"/api/public"},
			testPath:     "/api/private/users",
			want:         false,
		},
		{
			name:     "空配置",
			testPath: "/any/path",
			want:     false,
		},
		{
			name:         "前缀匹配边界情况 - 完全相同",
			skipPrefixes: []string{"/api"},
			testPath:     "/api",
			want:         true,
		},
		{
			name:         "前缀匹配边界情况 - 子路径",
			skipPrefixes: []string{"/api"},
			testPath:     "/api/v1/users",
			want:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewPathMatcher(tt.skipPaths, tt.skipPrefixes)
			got := matcher(tt.testPath)
			if got != tt.want {
				t.Errorf("NewPathMatcher() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldSkip(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		skipPaths    []string
		skipPrefixes []string
		want         bool
	}{
		{
			name:      "精确匹配",
			path:      "/health",
			skipPaths: []string{"/health", "/metrics"},
			want:      true,
		},
		{
			name:         "前缀匹配",
			path:         "/api/public/users",
			skipPrefixes: []string{"/api/public"},
			want:         true,
		},
		{
			name:         "混合匹配 - 精确优先",
			path:         "/health",
			skipPaths:    []string{"/health"},
			skipPrefixes: []string{"/api/public"},
			want:         true,
		},
		{
			name:         "都不匹配",
			path:         "/api/private/users",
			skipPaths:    []string{"/health"},
			skipPrefixes: []string{"/api/public"},
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkip(tt.path, tt.skipPaths, tt.skipPrefixes)
			if got != tt.want {
				t.Errorf("ShouldSkip() = %v, want %v", got, tt.want)
			}
		})
	}
}

// BenchmarkPathMatcher 性能对比测试
func BenchmarkNewPathMatcher(b *testing.B) {
	skipPaths := []string{"/health", "/metrics", "/version", "/ready", "/live"}
	skipPrefixes := []string{"/api/public", "/static", "/assets"}
	matcher := NewPathMatcher(skipPaths, skipPrefixes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher("/api/private/users")
	}
}

func BenchmarkShouldSkip(b *testing.B) {
	skipPaths := []string{"/health", "/metrics", "/version", "/ready", "/live"}
	skipPrefixes := []string{"/api/public", "/static", "/assets"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ShouldSkip("/api/private/users", skipPaths, skipPrefixes)
	}
}

// BenchmarkPathMatcherHit 测试命中场景的性能
func BenchmarkPathMatcherHit(b *testing.B) {
	skipPaths := []string{"/health", "/metrics", "/version", "/ready", "/live"}
	skipPrefixes := []string{"/api/public", "/static", "/assets"}
	matcher := NewPathMatcher(skipPaths, skipPrefixes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = matcher("/health") // 精确匹配命中
	}
}
