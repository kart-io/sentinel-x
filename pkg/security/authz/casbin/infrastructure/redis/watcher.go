package redis

import (
	"context"
	"sync"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/infra/pool"
	"github.com/redis/go-redis/v9"
)

// ChannelName is the Redis channel for policy updates.
const ChannelName = "casbin:policy:update"

// Watcher is the Redis watcher for Casbin.
type Watcher struct {
	client   *redis.Client
	channel  string
	callback func(string)
	pubsub   *redis.PubSub
	closeCh  chan struct{}
	wg       sync.WaitGroup
}

// NewWatcher creates a new Redis watcher
func NewWatcher(client *redis.Client, channel ...string) *Watcher {
	ch := ChannelName
	if len(channel) > 0 {
		ch = channel[0]
	}

	w := &Watcher{
		client:  client,
		channel: ch,
		closeCh: make(chan struct{}),
	}

	w.startSubscribe()
	return w
}

// recoverFromPanic recovers from panics in the subscription goroutine
func recoverFromPanic() {
	if r := recover(); r != nil {
		logger.Global().Errorw("Recovered from panic in Redis watcher subscription",
			"error", r,
			"component", "redis.watcher",
		)
	}
}

// handleChannelClosed handles the channel closure event
func handleChannelClosed(closeCh chan struct{}) {
	select {
	case <-closeCh:
		// Normal closure - watcher was explicitly closed
		logger.Global().Debugw("Redis subscription channel closed normally",
			"component", "redis.watcher",
		)
	default:
		// Abnormal closure - network error or other issue
		logger.Global().Warnw("Redis subscription channel closed unexpectedly",
			"component", "redis.watcher",
			"reason", "possible network disconnect or Redis error",
		)
	}
}

// executeCallback executes the callback in a separate goroutine to avoid blocking
// 使用 ants 池执行回调，避免无限制创建 goroutine
func executeCallback(callback func(string), payload string) {
	if callback == nil {
		return
	}

	task := func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Global().Errorw("Recovered from panic in callback execution",
					"error", r,
					"component", "redis.watcher",
					"payload", payload,
				)
			}
		}()

		callback(payload)
	}

	// 使用回调专用池执行
	if err := pool.SubmitToType(pool.CallbackPool, task); err != nil {
		// 池不可用时降级为直接执行 goroutine
		logger.Global().Warnw("failed to submit callback to pool, fallback to goroutine",
			"error", err.Error(),
			"component", "redis.watcher",
		)
		go task()
	}
}

func (w *Watcher) startSubscribe() {
	w.pubsub = w.client.Subscribe(context.Background(), w.channel)
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		defer recoverFromPanic()

		ch := w.pubsub.Channel()

		for {
			select {
			case <-w.closeCh:
				return
			case msg, ok := <-ch:
				if !ok {
					handleChannelClosed(w.closeCh)
					return
				}
				executeCallback(w.callback, msg.Payload)
			}
		}
	}()
}

// SetUpdateCallback sets the callback function to handle policy updates.
func (w *Watcher) SetUpdateCallback(callback func(string)) {
	w.callback = callback
}

// Update publishes a policy update message to Redis.
func (w *Watcher) Update() error {
	return w.client.Publish(context.Background(), ChannelName, "update").Err()
}

// Close closes the Redis watcher.
func (w *Watcher) Close() {
	if w.pubsub != nil {
		_ = w.pubsub.Close()
	}
	w.wg.Wait()
}
