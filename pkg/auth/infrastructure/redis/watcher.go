package redis

import (
	"context"
	"sync"

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

func (w *Watcher) startSubscribe() {
	w.pubsub = w.client.Subscribe(context.Background(), w.channel)
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()
		ch := w.pubsub.Channel()

		for {
			select {
			case <-w.closeCh:
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if w.callback != nil {
					w.callback(msg.Payload)
				}
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
		w.pubsub.Close()
	}
	w.wg.Wait()
}
