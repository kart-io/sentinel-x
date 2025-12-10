package redis

import (
	"context"
	"sync"

	"github.com/kart-io/logger"
	"github.com/redis/go-redis/v9"
)

const ChannelName = "casbin:policy:update"

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
func executeCallback(callback func(string), payload string) {
	if callback == nil {
		return
	}

	go func() {
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
	}()
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

func (w *Watcher) SetUpdateCallback(callback func(string)) {
	w.callback = callback
}

func (w *Watcher) Update() error {
	return w.client.Publish(context.Background(), w.channel, "update").Err()
}

func (w *Watcher) Close() {
	close(w.closeCh)
	if w.pubsub != nil {
		_ = w.pubsub.Close()
	}
	w.wg.Wait()
}
