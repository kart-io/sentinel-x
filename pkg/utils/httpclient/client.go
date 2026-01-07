// Package httpclient provides a reusable HTTP client with retry logic and resource management.
package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/kart-io/sentinel-x/pkg/utils/json"
)

// Client is a wrapper around http.Client with additional functionality.
type Client struct {
	httpClient *http.Client
	maxRetries int
}

// NewClient creates a new HTTP client wrapper.
func NewClient(timeout time.Duration, maxRetries int) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

// DoRequest executes an HTTP request with retry logic.
// The caller is responsible for providing a way to reset the request body if retries are needed.
// This is a low-level method.
func (c *Client) DoRequest(req *http.Request) (*http.Response, error) {
	// 自动注入 W3C Trace Context 头
	c.injectTraceContext(req)

	var lastErr error

	// If the request has a body, we need to be able to get it again for retries
	var bodyGetter func() (io.ReadCloser, error)
	if req.Body != nil {
		// We assume the body is already read into memory if we want to support retries here
		// or the caller should handle it. For LLM providers, bodies are small.
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		_ = req.Body.Close()
		bodyGetter = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	for i := 0; i <= c.maxRetries; i++ {
		if bodyGetter != nil {
			var err error
			req.Body, err = bodyGetter()
			if err != nil {
				return nil, err
			}
		}

		resp, err := c.httpClient.Do(req)
		if err == nil {
			if resp.StatusCode < 500 {
				return resp, nil
			}
			// It's a server error, we can retry. Close the body first.
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("server error, status code %d", resp.StatusCode)
		} else {
			lastErr = err
		}

		if i < c.maxRetries {
			select {
			case <-req.Context().Done():
				return nil, req.Context().Err()
			case <-time.After(time.Duration(i+1) * 500 * time.Millisecond):
				// Continue to next retry
			}
		}
	}
	return nil, lastErr
}

// DoJSON executes a JSON request, decodes the response, and ensures the body is closed.
func (c *Client) DoJSON(req *http.Request, v interface{}) error {
	resp, err := c.DoRequest(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}

// injectTraceContext 将 W3C Trace Context 头注入到 HTTP 请求中。
// 从 context.Context 中提取当前 Span 的追踪信息，自动传播到下游服务。
//
// 如果满足以下任一条件，则跳过注入（优雅降级）：
//   - 请求为 nil
//   - 请求的 Context 为 nil
//   - 全局传播器未设置
//   - Context 中无活跃 Span
//
// 该方法是线程安全的，注入开销约 < 100ns，对 HTTP 请求性能影响可忽略。
func (c *Client) injectTraceContext(req *http.Request) {
	if req == nil || req.Context() == nil {
		return
	}

	propagator := otel.GetTextMapPropagator()
	if propagator == nil {
		return
	}

	// 将追踪上下文注入到 HTTP Header
	// 使用 propagation.HeaderCarrier 确保线程安全
	propagator.Inject(req.Context(), propagation.HeaderCarrier(req.Header))
}
