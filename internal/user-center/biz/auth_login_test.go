package biz

import (
	"context"
	"testing"

	"github.com/kart-io/sentinel-x/internal/model"
	"github.com/kart-io/sentinel-x/internal/user-center/store"
	"github.com/kart-io/sentinel-x/pkg/component/redis"
	redisopts "github.com/kart-io/sentinel-x/pkg/options/redis"
	"github.com/kart-io/sentinel-x/pkg/security/auth/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(&model.User{}, &model.LoginLog{})
	require.NoError(t, err)
	return db
}

func setupTestRedis(t *testing.T) *redis.Client {
    // Attempt local connection
	opts := redisopts.NewOptions()
    opts.Host = "127.0.0.1"
    opts.Port = 6379

    client, err := redis.New(opts)
    if err != nil {
        t.Skipf("Skipping redis dependent tests: %v", err)
    }
    // Fail fast if redis is not reachable
    if err := client.Client().Ping(context.Background()).Err(); err != nil {
        t.Skipf("Skipping redis dependent tests: failed to ping redis: %v", err)
    }
    return client
}

func setupAuthService(t *testing.T, db *gorm.DB, rdb *redis.Client) *AuthService {
    userStore := store.NewUserStore(db)
    logStore := store.NewLogStore(db)

	// JWT Setup
	jwtOpts := jwt.NewOptions()
	jwtOpts.Key = "secret-key-must-be-32-bytes-long-aaa"
    jwtStore := jwt.NewRedisStore(rdb, "jwt:test:")
    jwtAuth, err := jwt.New(jwt.WithOptions(jwtOpts), jwt.WithStore(jwtStore))
    require.NoError(t, err)

    return NewAuthService(jwtAuth, userStore, logStore, rdb)
}

func TestAuthService_FullLoginFlow(t *testing.T) {
    db := setupTestDB(t)
    rdb := setupTestRedis(t)
    svc := setupAuthService(t, db, rdb)
    ctx := context.Background()

    // 1. Create Test User
    password := "password123"
    hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    user := &model.User{
        Username: "test_login",
        Password: string(hashed),
        Status:   1,
    }
    db.Create(user)

    // Clear Redis keys
    rdb.Client().Del(ctx, "login_fail:test_login", "login_lock:test_login")

    t.Run("Login Success", func(t *testing.T) {
        req := &model.LoginRequest{
            Username: "test_login",
            Password: password,
        }
        resp, err := svc.Login(ctx, req, "test-agent", "127.0.0.1")
        require.NoError(t, err)
        assert.NotEmpty(t, resp.Token)
        assert.NotEmpty(t, resp.RefreshToken)

        // Verify Log
        var log model.LoginLog
        db.Last(&log)
        assert.Equal(t, "test_login", log.Username)
        assert.Equal(t, 1, log.Status)
    })

    t.Run("Refresh Token", func(t *testing.T) {
        // Get valid refresh token
        req := &model.LoginRequest{
            Username: "test_login",
            Password: password,
        }
        resp, _ := svc.Login(ctx, req, "test-agent", "127.0.0.1")

        // Refresh
        refreshResp, err := svc.RefreshToken(ctx, resp.RefreshToken)
        require.NoError(t, err)
        assert.NotEmpty(t, refreshResp.Token)
        assert.NotEqual(t, resp.Token, refreshResp.Token)
    })

    t.Run("Captcha Mock", func(t *testing.T) {
        id, _, err := svc.GetCaptcha(ctx)
        require.NoError(t, err)
        assert.Equal(t, "mock-captcha-id", id)

        valid := svc.VerifyCaptcha(ctx, id, "1234")
        assert.True(t, valid)
    })
}

func TestAuthService_Lockout(t *testing.T) {
    db := setupTestDB(t)
    rdb := setupTestRedis(t)
    svc := setupAuthService(t, db, rdb)
    ctx := context.Background()

    // Create User
    password := "password123"
    hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    user := &model.User{
        Username: "test_lockout",
        Password: string(hashed),
        Status:   1,
    }
    db.Create(user)

    // Clear Redis
    rdb.Client().Del(ctx, "login_fail:test_lockout", "login_lock:test_lockout")

    // Fail 5 times
    req := &model.LoginRequest{
        Username: "test_lockout",
        Password: "wrong_password",
    }

    for i := 0; i < 5; i++ {
        _, err := svc.Login(ctx, req, "agent", "ip")
        assert.Error(t, err)
    }

    // 6th attempt should be locked
    _, err := svc.Login(ctx, req, "agent", "ip")
    require.Error(t, err)
    // Check if error message contains "locked"
    // Note: implementation returns errors.ErrTooManyRequests
    assert.Contains(t, err.Error(), "锁定")

    // Even correct password should fail now
    req.Password = password
    _, err = svc.Login(ctx, req, "agent", "ip")
    require.Error(t, err)
     assert.Contains(t, err.Error(), "锁定")
}
