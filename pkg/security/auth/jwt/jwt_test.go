package jwt

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kart-io/sentinel-x/pkg/security/auth"
	"github.com/kart-io/sentinel-x/pkg/utils/errors"
)

// testKey is a secure test key that meets the minimum 64 character requirement
const (
	testKey  = "test-secret-key-at-least-64-chars-long-for-security-purposes!!!!"
	testUser = "user-123"
)

// createTestJWT 创建一个用于测试的JWT实例
func createTestJWT(t *testing.T, opts ...Option) *JWT {
	t.Helper()

	// 默认选项
	defaultOpts := []Option{
		WithKey(testKey),
		WithSigningMethod("HS256"),
		WithExpired(time.Hour),
		WithIssuer("test-issuer"),
	}

	// 合并自定义选项
	allOpts := make([]Option, 0, len(defaultOpts)+len(opts))
	allOpts = append(allOpts, defaultOpts...)
	allOpts = append(allOpts, opts...)

	j, err := New(allOpts...)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	return j
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		opts    []Option
		wantErr bool
	}{
		{
			name: "Valid HMAC configuration",
			opts: []Option{
				WithKey(testKey),
				WithSigningMethod("HS256"),
				WithExpired(time.Hour),
			},
			wantErr: false,
		},
		{
			name: "Valid HS384",
			opts: []Option{
				WithKey(testKey),
				WithSigningMethod("HS384"),
			},
			wantErr: false,
		},
		{
			name: "Valid HS512",
			opts: []Option{
				WithKey(testKey),
				WithSigningMethod("HS512"),
			},
			wantErr: false,
		},
		{
			name: "Key too short",
			opts: []Option{
				WithKey("short"),
				WithSigningMethod("HS256"),
			},
			wantErr: true,
		},
		{
			name: "Invalid signing method",
			opts: []Option{
				WithKey(testKey),
				WithSigningMethod("INVALID"),
			},
			wantErr: true,
		},
		{
			name: "Missing key",
			opts: []Option{
				WithSigningMethod("HS256"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJWT_Type(t *testing.T) {
	j := createTestJWT(t)

	if got := j.Type(); got != "jwt" {
		t.Errorf("Type() = %v, want %v", got, "jwt")
	}
}

func TestJWT_Sign(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	tests := []struct {
		name    string
		subject string
		opts    []auth.SignOption
		wantErr bool
	}{
		{
			name:    "Basic sign",
			subject: testUser,
			opts:    nil,
			wantErr: false,
		},
		{
			name:    "Sign with extra claims",
			subject: "user-456",
			opts: []auth.SignOption{
				auth.WithExtra(map[string]interface{}{
					"role":  "admin",
					"email": "test@example.com",
				}),
			},
			wantErr: false,
		},
		{
			name:    "Sign with audience",
			subject: "user-789",
			opts: []auth.SignOption{
				auth.WithAudience("api", "web"),
			},
			wantErr: false,
		},
		{
			name:    "Sign with custom expiration",
			subject: "user-999",
			opts: []auth.SignOption{
				auth.WithExpiresAt(time.Now().Add(30 * time.Minute)),
			},
			wantErr: false,
		},
		{
			name:    "Sign with custom token ID",
			subject: "user-custom",
			opts: []auth.SignOption{
				auth.WithTokenID("custom-token-id-123"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := j.Sign(ctx, tt.subject, tt.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if token.GetAccessToken() == "" {
					t.Error("Expected non-empty access token")
				}
				if token.GetTokenType() != "Bearer" {
					t.Errorf("Expected token type 'Bearer', got '%s'", token.GetTokenType())
				}
				if token.GetExpiresAt() <= time.Now().Unix() {
					t.Error("Expected future expiration time")
				}
				if token.GetExpiresIn() <= 0 {
					t.Error("Expected positive expires_in")
				}
			}
		})
	}
}

func TestJWT_SignAndVerify(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	// 签名令牌
	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 验证令牌
	claims, err := j.Verify(ctx, token.GetAccessToken())
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	// 检查声明
	if claims.Subject != testUser {
		t.Errorf("Expected subject 'user-123', got '%s'", claims.Subject)
	}
	if claims.Issuer != "test-issuer" {
		t.Errorf("Expected issuer 'test-issuer', got '%s'", claims.Issuer)
	}
	if claims.ID == "" {
		t.Error("Expected non-empty token ID")
	}
}

func TestJWT_SignAndVerifyWithExtraClaims(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	extra := map[string]interface{}{
		"role":  "admin",
		"email": "admin@example.com",
		"level": float64(5),
	}

	// 签名带额外声明的令牌
	token, err := j.Sign(ctx, "admin-user", auth.WithExtra(extra))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 验证令牌
	claims, err := j.Verify(ctx, token.GetAccessToken())
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	// 检查额外声明
	if claims.Extra == nil {
		t.Fatal("Expected extra claims")
	}

	if role := claims.GetExtraString("role"); role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}

	if email := claims.GetExtraString("email"); email != "admin@example.com" {
		t.Errorf("Expected email 'admin@example.com', got '%s'", email)
	}

	if level, ok := claims.GetExtra("level"); !ok || level != float64(5) {
		t.Errorf("Expected level 5, got %v", level)
	}
}

func TestJWT_Verify_EmptyToken(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	_, err := j.Verify(ctx, "")
	if err == nil {
		t.Error("Expected error for empty token")
	}

	if !errors.IsCode(err, errors.ErrInvalidToken.Code) {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestJWT_Verify_MalformedToken(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "Random string",
			token: "not-a-valid-token",
		},
		{
			name:  "Incomplete JWT",
			token: "header.payload",
		},
		{
			name:  "Invalid base64",
			token: "!!!.###.$$$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := j.Verify(ctx, tt.token)
			if err == nil {
				t.Error("Expected error for malformed token")
			}
			if !errors.IsCode(err, errors.ErrInvalidToken.Code) {
				t.Errorf("Expected ErrInvalidToken, got %v", err)
			}
		})
	}
}

func TestJWT_Verify_InvalidSignature(t *testing.T) {
	ctx := context.Background()

	// 使用第一个密钥创建JWT
	j1 := createTestJWT(t, WithKey(testKey))

	// 使用不同密钥创建另一个JWT
	j2 := createTestJWT(t, WithKey("different-secret-key-64-chars-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"))

	// 使用j2签名
	token, err := j2.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 尝试使用j1验证（不同的密钥）
	_, err = j1.Verify(ctx, token.GetAccessToken())
	if err == nil {
		t.Error("Expected error for invalid signature")
	}

	if !errors.IsCode(err, errors.ErrInvalidToken.Code) {
		t.Errorf("Expected ErrInvalidToken, got %v", err)
	}
}

func TestJWT_Verify_ExpiredToken(t *testing.T) {
	ctx := context.Background()

	// 创建一个立即过期的JWT
	j := createTestJWT(t, WithExpired(1*time.Millisecond))

	// 签名令牌
	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 等待令牌过期
	time.Sleep(10 * time.Millisecond)

	// 尝试验证过期令牌
	_, err = j.Verify(ctx, token.GetAccessToken())
	if err == nil {
		t.Error("Expected error for expired token")
	}

	if !errors.IsCode(err, errors.ErrTokenExpired.Code) {
		t.Errorf("Expected ErrTokenExpired, got %v", err)
	}
}

func TestJWT_Refresh(t *testing.T) {
	ctx := context.Background()

	// 创建配置
	// 创建配置
	opts := &Options{
		Key:           testKey,
		SigningMethod: "HS256",
		Expired:       1 * time.Second, // 令牌有效期 1秒
		MaxRefresh:    5 * time.Hour,   // 最大刷新窗口
		Issuer:        "test-issuer",
	}

	j, err := New(WithOptions(opts))
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// 签名原始令牌
	originalToken, err := j.Sign(ctx, testUser, auth.WithExtra(map[string]interface{}{
		"role": "user",
	}))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 等待令牌过期
	time.Sleep(1100 * time.Millisecond)

	// 刷新令牌（原令牌已过期但在 MaxRefresh 窗口内）
	newToken, err := j.Refresh(ctx, originalToken.GetAccessToken())
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if newToken.GetAccessToken() == "" {
		t.Error("Expected non-empty refreshed token")
	}

	// 验证新令牌是有效的（刚刚生成的）
	claims, err := j.Verify(ctx, newToken.GetAccessToken())
	if err != nil {
		t.Fatalf("Failed to verify refreshed token: %v", err)
	}

	// 检查声明是否保留
	if claims.Subject != testUser {
		t.Errorf("Expected subject 'user-123', got '%s'", claims.Subject)
	}

	if role := claims.GetExtraString("role"); role != "user" {
		t.Errorf("Expected role 'user', got '%s'", role)
	}
}

func TestJWT_Refresh_ExceedsMaxRefresh(t *testing.T) {
	ctx := context.Background()

	// MaxRefresh 设置为很短时间
	j := createTestJWT(t,
		WithExpired(1*time.Millisecond),
		WithOptions(&Options{
			Key:           testKey,
			SigningMethod: "HS256",
			Expired:       1 * time.Millisecond,
			MaxRefresh:    10 * time.Millisecond,
			Issuer:        "test-issuer",
		}),
	)

	// 签名令牌
	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 等待超过 MaxRefresh 时间
	time.Sleep(15 * time.Millisecond)

	// 尝试刷新（应该失败）
	_, err = j.Refresh(ctx, token.GetAccessToken())
	if err == nil {
		t.Error("Expected error when refresh window exceeded")
	}

	if !errors.IsCode(err, errors.ErrSessionExpired.Code) {
		t.Errorf("Expected ErrSessionExpired, got %v", err)
	}
}

func TestJWT_Revoke(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	j := createTestJWT(t, WithStore(store))

	// 签名令牌
	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	tokenStr := token.GetAccessToken()

	// 验证令牌有效
	_, err = j.Verify(ctx, tokenStr)
	if err != nil {
		t.Fatalf("Token should be valid before revocation: %v", err)
	}

	// 撤销令牌
	err = j.Revoke(ctx, tokenStr)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// 尝试验证已撤销的令牌
	_, err = j.Verify(ctx, tokenStr)
	if err == nil {
		t.Error("Expected error for revoked token")
	}

	if !errors.IsCode(err, errors.ErrTokenRevoked.Code) {
		t.Errorf("Expected ErrTokenRevoked, got %v", err)
	}
}

func TestJWT_Revoke_WithoutStore(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t) // 没有 store

	// 签名令牌
	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 尝试撤销（应该失败）
	err = j.Revoke(ctx, token.GetAccessToken())
	if err == nil {
		t.Error("Expected error when revoking without store")
	}

	if !errors.IsCode(err, errors.ErrNotImplemented.Code) {
		t.Errorf("Expected ErrNotImplemented, got %v", err)
	}
}

func TestJWT_Revoke_ExpiredToken(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	defer func() { _ = store.Close() }()

	j := createTestJWT(t,
		WithStore(store),
		WithExpired(1*time.Millisecond),
	)

	// 签名令牌
	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 等待令牌过期
	time.Sleep(10 * time.Millisecond)

	// 撤销已过期的令牌（应该成功，因为已经无效了）
	err = j.Revoke(ctx, token.GetAccessToken())
	if err != nil {
		t.Errorf("Should allow revoking expired token: %v", err)
	}
}

func TestJWT_WithDifferentSigningMethods(t *testing.T) {
	ctx := context.Background()

	methods := []string{"HS256", "HS384", "HS512"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			j := createTestJWT(t, WithSigningMethod(method))

			// 签名
			token, err := j.Sign(ctx, testUser)
			if err != nil {
				t.Fatalf("Failed to sign with %s: %v", method, err)
			}

			// 验证
			claims, err := j.Verify(ctx, token.GetAccessToken())
			if err != nil {
				t.Fatalf("Failed to verify with %s: %v", method, err)
			}

			if claims.Subject != testUser {
				t.Errorf("Subject mismatch with %s", method)
			}
		})
	}
}

func TestJWT_WithAudience(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	// 签名带 audience 的令牌
	token, err := j.Sign(ctx, testUser, auth.WithAudience("api", "web", "mobile"))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 验证
	claims, err := j.Verify(ctx, token.GetAccessToken())
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	// 检查 audience
	expectedAud := []string{"api", "web", "mobile"}
	if len(claims.Audience) != len(expectedAud) {
		t.Errorf("Expected %d audiences, got %d", len(expectedAud), len(claims.Audience))
	}

	for i, aud := range expectedAud {
		if claims.Audience[i] != aud {
			t.Errorf("Expected audience[%d] = %s, got %s", i, aud, claims.Audience[i])
		}
	}
}

func TestJWT_TokenValidation(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	// 创建一个令牌
	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 验证令牌
	claims, err := j.Verify(ctx, token.GetAccessToken())
	if err != nil {
		t.Fatalf("Failed to verify token: %v", err)
	}

	// 检查 claims 是否有效
	if !claims.Valid() {
		t.Error("Claims should be valid")
	}

	if claims.IsExpired() {
		t.Error("Claims should not be expired")
	}
}

func TestJWT_ConcurrentSignAndVerify(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	const numGoroutines = 100
	done := make(chan bool, numGoroutines)

	// 并发签名和验证
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// 签名
			token, err := j.Sign(ctx, "user-"+string(rune(id)))
			if err != nil {
				t.Errorf("Concurrent sign failed: %v", err)
				return
			}

			// 验证
			_, err = j.Verify(ctx, token.GetAccessToken())
			if err != nil {
				t.Errorf("Concurrent verify failed: %v", err)
			}
		}(i)
	}

	// 等待所有 goroutine 完成
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestJWT_SigningMethodMismatch(t *testing.T) {
	ctx := context.Background()

	// 使用 HS256 创建JWT
	j256 := createTestJWT(t, WithSigningMethod("HS256"))

	// 使用 HS384 创建JWT
	j384 := createTestJWT(t, WithSigningMethod("HS384"))

	// 使用 HS256 签名
	token, err := j256.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign with HS256: %v", err)
	}

	// 尝试用 HS384 验证（应该失败）
	_, err = j384.Verify(ctx, token.GetAccessToken())
	if err == nil {
		t.Error("Expected error for signing method mismatch")
	}

	if !strings.Contains(err.Error(), "unexpected signing method") {
		t.Errorf("Expected signing method error, got: %v", err)
	}
}

func TestJWT_TokenJSON(t *testing.T) {
	ctx := context.Background()
	j := createTestJWT(t)

	token, err := j.Sign(ctx, testUser)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// 测试 JSON 序列化
	jsonBytes, err := token.JSON()
	if err != nil {
		t.Fatalf("Failed to serialize token to JSON: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Error("Expected non-empty JSON bytes")
	}

	// 测试 Map 转换
	tokenMap := token.Map()
	if tokenMap == nil {
		t.Fatal("Expected non-nil token map")
	}

	if tokenMap["access_token"] == "" {
		t.Error("Expected access_token in map")
	}
	if tokenMap["token_type"] != "Bearer" {
		t.Error("Expected token_type 'Bearer' in map")
	}
}
