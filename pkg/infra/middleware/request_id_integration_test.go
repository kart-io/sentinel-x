package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kart-io/sentinel-x/pkg/infra/server/transport"
	mwopts "github.com/kart-io/sentinel-x/pkg/options/middleware"
)

// TestRequestIDWithOptions_ULIDGenerator 测试使用 ULID 生成器的配置
func TestRequestIDWithOptions_ULIDGenerator(t *testing.T) {
	opts := mwopts.RequestIDOptions{
		Header:        "X-Request-ID",
		GeneratorType: "ulid",
	}
	middleware := RequestIDWithOptions(opts, nil)

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	ctx := newMockContext(req, w)

	handler(ctx)

	// 验证响应头包含 request ID
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("Request ID should be set in response header")
	}

	// 验证 ULID 长度 (26 字符)
	if len(requestID) != 26 {
		t.Errorf("ULID should be 26 characters, got %d: %s", len(requestID), requestID)
	}

	// 验证 request ID 存储在 context 中
	storedID := GetRequestID(ctx.Request())
	if storedID != requestID {
		t.Errorf("Request ID mismatch: expected %s, got %s", requestID, storedID)
	}
}

// TestRequestIDWithOptions_RandomHexGenerator 测试使用随机十六进制生成器的配置
func TestRequestIDWithOptions_RandomHexGenerator(t *testing.T) {
	tests := []struct {
		name          string
		generatorType string
		expectedLen   int
	}{
		{"Random 类型", "random", 32},
		{"Hex 类型", "hex", 32},
		{"空字符串(默认)", "", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := mwopts.RequestIDOptions{
				Header:        "X-Request-ID",
				GeneratorType: tt.generatorType,
			}
			middleware := RequestIDWithOptions(opts, nil)

			handler := middleware(func(c transport.Context) {
				c.JSON(200, map[string]string{"status": "ok"})
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			ctx := newMockContext(req, w)

			handler(ctx)

			requestID := w.Header().Get("X-Request-ID")
			if requestID == "" {
				t.Error("Request ID should be set in response header")
			}

			if len(requestID) != tt.expectedLen {
				t.Errorf("Expected length %d, got %d: %s", tt.expectedLen, len(requestID), requestID)
			}
		})
	}
}

// TestRequestIDWithOptions_GeneratorPerformance 测试不同生成器的性能特征
func TestRequestIDWithOptions_GeneratorPerformance(t *testing.T) {
	tests := []struct {
		name          string
		generatorType string
	}{
		{"Random", "random"},
		{"ULID", "ulid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := mwopts.RequestIDOptions{
				Header:        "X-Request-ID",
				GeneratorType: tt.generatorType,
			}
			middleware := RequestIDWithOptions(opts, nil)

			handler := middleware(func(c transport.Context) {
				c.JSON(200, map[string]string{"status": "ok"})
			})

			// 生成多个 ID 验证唯一性
			seen := make(map[string]bool)
			for i := 0; i < 1000; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				ctx := newMockContext(req, w)

				handler(ctx)

				requestID := w.Header().Get("X-Request-ID")
				if seen[requestID] {
					t.Errorf("Duplicate request ID found: %s", requestID)
				}
				seen[requestID] = true
			}

			t.Logf("%s 生成器: 生成了 %d 个唯一 ID", tt.name, len(seen))
		})
	}
}

// TestRequestIDWithOptions_ULIDSortability 测试 ULID 的时间可排序性
func TestRequestIDWithOptions_ULIDSortability(t *testing.T) {
	opts := mwopts.RequestIDOptions{
		Header:        "X-Request-ID",
		GeneratorType: "ulid",
	}
	middleware := RequestIDWithOptions(opts, nil)

	handler := middleware(func(c transport.Context) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	// 连续生成多个 ID
	var ids []string
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()
		ctx := newMockContext(req, w)

		handler(ctx)

		requestID := w.Header().Get("X-Request-ID")
		ids = append(ids, requestID)
	}

	// 验证 ID 是递增的 (词典序)
	for i := 1; i < len(ids); i++ {
		if ids[i] <= ids[i-1] {
			t.Errorf("ULID 应该是递增的,但 ids[%d](%s) <= ids[%d](%s)",
				i, ids[i], i-1, ids[i-1])
		}
	}

	t.Logf("验证了 %d 个 ULID 的时间可排序性", len(ids))
}

// TestRequestIDOptions_Validation 测试配置验证
func TestRequestIDOptions_Validation(t *testing.T) {
	tests := []struct {
		name      string
		opts      mwopts.RequestIDOptions
		wantError bool
	}{
		{
			name: "有效配置 - ULID",
			opts: mwopts.RequestIDOptions{
				Header:        "X-Request-ID",
				GeneratorType: "ulid",
			},
			wantError: false,
		},
		{
			name: "有效配置 - Random",
			opts: mwopts.RequestIDOptions{
				Header:        "X-Request-ID",
				GeneratorType: "random",
			},
			wantError: false,
		},
		{
			name: "无效配置 - 空 Header",
			opts: mwopts.RequestIDOptions{
				Header:        "",
				GeneratorType: "ulid",
			},
			wantError: true,
		},
		{
			name: "无效配置 - 未知生成器类型",
			opts: mwopts.RequestIDOptions{
				Header:        "X-Request-ID",
				GeneratorType: "unknown",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.opts.Validate()
			hasError := len(errs) > 0

			if hasError != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v, errors: %v", hasError, tt.wantError, errs)
			}
		})
	}
}
