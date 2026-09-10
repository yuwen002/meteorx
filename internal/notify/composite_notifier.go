package notify

import (
	"context"
	"sync"
	"time"

	"meteorx/pkg/logger"
)

// CompositeNotifier 组合通知器
// 一条消息同时发送到多个渠道
type CompositeNotifier struct {
	mu       sync.RWMutex
	channels map[ChannelType]Notifier
}

// NewCompositeNotifier 创建组合通知器
func NewCompositeNotifier() *CompositeNotifier {
	return &CompositeNotifier{
		channels: make(map[ChannelType]Notifier),
	}
}

// AddChannel 注册通知渠道
func (c *CompositeNotifier) AddChannel(n Notifier) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n != nil {
		c.channels[n.Type()] = n
		logger.Infof("[Notify] Registered channel: %s (%s)", n.Name(), n.Type())
	}
}

// RemoveChannel 移除通知渠道
func (c *CompositeNotifier) RemoveChannel(channel ChannelType) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channel)
}

// Send 发送消息到指定渠道
func (c *CompositeNotifier) Send(ctx context.Context, msg *Message) []SendResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var targets []Notifier
	if msg.Channel == "" {
		for _, n := range c.channels {
			targets = append(targets, n)
		}
	} else if n, ok := c.channels[msg.Channel]; ok {
		targets = []Notifier{n}
	} else {
		return []SendResult{{
			Channel: msg.Channel,
			Success: false,
			Error:   &NotifyError{Channel: msg.Channel, MsgID: msg.ID, Err: errChannelNotFound},
		}}
	}

	results := make([]SendResult, 0, len(targets))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, n := range targets {
		wg.Add(1)
		go func(notifier Notifier) {
			defer wg.Done()
			start := time.Now()
			err := notifier.Send(ctx, msg)
			latency := time.Since(start)

			mu.Lock()
			results = append(results, SendResult{
				Channel: notifier.Type(),
				Success: err == nil,
				Error:   err,
				Latency: latency,
			})
			mu.Unlock()

			if err != nil {
				logger.Warnf("[Notify] %s send failed: %v (latency: %v)", notifier.Name(), err, latency)
			} else {
				logger.Debugf("[Notify] %s sent successfully (latency: %v)", notifier.Name(), latency)
			}
		}(n)
	}

	wg.Wait()
	return results
}

// SendToAll 发送消息到所有已注册渠道
func (c *CompositeNotifier) SendToAll(ctx context.Context, msg *Message) []SendResult {
	msg.Channel = ""
	return c.Send(ctx, msg)
}

// SendToChannel 发送消息到指定渠道
func (c *CompositeNotifier) SendToChannel(ctx context.Context, channel ChannelType, msg *Message) []SendResult {
	msg.Channel = channel
	return c.Send(ctx, msg)
}

// HasChannel 检查指定渠道是否已注册
func (c *CompositeNotifier) HasChannel(channel ChannelType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.channels[channel]
	return ok
}

// ChannelCount 返回已注册渠道数量
func (c *CompositeNotifier) ChannelCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.channels)
}

var errChannelNotFound = &channelError{msg: "channel not registered"}

type channelError struct{ msg string }

func (e *channelError) Error() string { return e.msg }
