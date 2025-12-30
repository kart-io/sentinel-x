package biz

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// benchmarkPassword 基准测试使用的密码常量
const benchmarkPassword = "testPassword123!"

// BenchmarkPasswordHashing 测试密码哈希性能
func BenchmarkPasswordHashing(b *testing.B) {
	password := benchmarkPassword

	b.Run("DefaultCost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		}
	})

	b.Run("MinCost", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		}
	})
}

// BenchmarkPasswordVerification 测试密码验证性能
func BenchmarkPasswordVerification(b *testing.B) {
	password := benchmarkPassword
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	}
}

// BenchmarkPasswordVerificationParallel 并行密码验证性能测试
func BenchmarkPasswordVerificationParallel(b *testing.B) {
	password := benchmarkPassword
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
		}
	})
}
