package pool

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	p, err := NewPool("test", DefaultPoolConfig())
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	defer p.Release()

	if p.Name() != "test" {
		t.Errorf("池名称不匹配: 期望 test, 实际 %s", p.Name())
	}

	if p.Cap() != 1000 {
		t.Errorf("池容量不匹配: 期望 1000, 实际 %d", p.Cap())
	}
}

func TestPoolSubmit(t *testing.T) {
	p, err := NewPool("test", &PoolConfig{
		Capacity:       10,
		ExpiryDuration: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	defer p.Release()

	var counter atomic.Int32
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		err := p.Submit(func() {
			defer wg.Done()
			counter.Add(1)
		})
		if err != nil {
			t.Errorf("提交任务失败: %v", err)
			wg.Done()
		}
	}

	wg.Wait()

	if counter.Load() != 100 {
		t.Errorf("任务执行数不匹配: 期望 100, 实际 %d", counter.Load())
	}
}

func TestPoolSubmitWithContext(t *testing.T) {
	p, err := NewPool("test", &PoolConfig{
		Capacity:       5,
		ExpiryDuration: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	defer p.Release()

	// 测试正常执行
	var executed atomic.Bool
	ctx := context.Background()
	err = p.SubmitWithContext(ctx, func() {
		executed.Store(true)
	})
	if err != nil {
		t.Errorf("提交任务失败: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if !executed.Load() {
		t.Error("任务未执行")
	}

	// 测试已取消的上下文
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel() // 立即取消

	err = p.SubmitWithContext(canceledCtx, func() {
		t.Error("已取消的上下文不应执行任务")
	})
	if err != context.Canceled {
		t.Errorf("期望 context.Canceled 错误, 实际: %v", err)
	}
}

func TestPoolPanicRecovery(t *testing.T) {
	var panicCaught atomic.Bool

	p, err := NewPool("test", &PoolConfig{
		Capacity:       5,
		ExpiryDuration: 5 * time.Second,
		PanicHandler: func(r interface{}) {
			panicCaught.Store(true)
		},
	})
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	defer p.Release()

	err = p.Submit(func() {
		panic("测试 panic")
	})
	if err != nil {
		t.Errorf("提交任务失败: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if !panicCaught.Load() {
		t.Error("panic 未被捕获")
	}
}

func TestPoolClosed(t *testing.T) {
	p, err := NewPool("test", &PoolConfig{
		Capacity:       5,
		ExpiryDuration: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}

	p.Release()

	err = p.Submit(func() {
		t.Error("已关闭的池不应执行任务")
	})
	if err != ErrPoolClosed {
		t.Errorf("期望 ErrPoolClosed, 实际: %v", err)
	}
}

func TestManager(t *testing.T) {
	mgr := NewManager()
	defer func() {
		_ = mgr.Close()
	}()

	// 注册池
	err := mgr.Register("test-pool", &PoolConfig{
		Capacity:       10,
		ExpiryDuration: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("注册池失败: %v", err)
	}

	// 重复注册
	err = mgr.Register("test-pool", DefaultPoolConfig())
	if err == nil {
		t.Error("重复注册应返回错误")
	}

	// 获取池
	p, err := mgr.Get("test-pool")
	if err != nil {
		t.Errorf("获取池失败: %v", err)
	}
	if p == nil {
		t.Error("池不应为 nil")
	}

	// 获取不存在的池
	_, err = mgr.Get("non-existent")
	if err == nil {
		t.Error("获取不存在的池应返回错误")
	}

	// 提交任务
	var executed atomic.Bool
	err = mgr.Submit("test-pool", func() {
		executed.Store(true)
	})
	if err != nil {
		t.Errorf("提交任务失败: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if !executed.Load() {
		t.Error("任务未执行")
	}

	// 列表
	list := mgr.List()
	if len(list) != 1 {
		t.Errorf("池列表长度不匹配: 期望 1, 实际 %d", len(list))
	}

	// 统计
	stats := mgr.Stats()
	if len(stats) != 1 {
		t.Errorf("统计信息长度不匹配: 期望 1, 实际 %d", len(stats))
	}
}

func TestGlobalPool(t *testing.T) {
	// 重置全局状态
	ResetGlobal()

	// 初始化
	err := InitGlobal()
	if err != nil {
		t.Fatalf("初始化全局池失败: %v", err)
	}
	defer func() {
		_ = CloseGlobal()
	}()
	// 获取全局管理器
	mgr := GetGlobal()
	if mgr == nil {
		t.Fatal("全局管理器不应为 nil")
	}

	// 检查预定义池
	pools := mgr.List()
	expectedPools := []string{
		string(DefaultPool),
		string(HealthCheckPool),
		string(BackgroundPool),
		string(CallbackPool),
		string(TimeoutPool),
	}

	if len(pools) != len(expectedPools) {
		t.Errorf("预定义池数量不匹配: 期望 %d, 实际 %d", len(expectedPools), len(pools))
	}

	// 提交任务
	var executed atomic.Bool
	err = Submit(func() {
		executed.Store(true)
	})
	if err != nil {
		t.Errorf("提交任务失败: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	if !executed.Load() {
		t.Error("任务未执行")
	}
}

func TestPoolNonblocking(t *testing.T) {
	p, err := NewPool("test", &PoolConfig{
		Capacity:       1,
		ExpiryDuration: 5 * time.Second,
		Nonblocking:    true,
	})
	if err != nil {
		t.Fatalf("创建池失败: %v", err)
	}
	defer p.Release()

	// 占用唯一的 worker
	done := make(chan struct{})
	err = p.Submit(func() {
		<-done
	})
	if err != nil {
		t.Errorf("提交任务失败: %v", err)
	}

	// 尝试提交更多任务（应失败）
	err = p.Submit(func() {
		t.Error("非阻塞模式下池满时不应执行任务")
	})
	if err == nil {
		t.Error("非阻塞模式下池满时应返回错误")
	}

	close(done)
}

func BenchmarkPoolSubmit(b *testing.B) {
	p, _ := NewPool("bench", &PoolConfig{
		Capacity:       1000,
		ExpiryDuration: 5 * time.Second,
		PreAlloc:       true,
	})
	defer p.Release()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = p.Submit(func() {
				// 模拟简单任务
			})
		}
	})
}

func BenchmarkDirectGoroutine(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			go func() {
				// 模拟简单任务
			}()
		}
	})
}
