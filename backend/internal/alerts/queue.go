package alerts

import (
	"context"
	"log/slog"
)

// Queue is an async dispatch queue for alert delivery.
// Callers enqueue alerts without blocking; a background goroutine consumes and delivers.
type Queue struct {
	cfg Config
	ch  chan Alert
}

// NewQueue creates a Queue with the given configuration and a 100-item buffer.
func NewQueue(cfg Config) *Queue {
	return &Queue{
		cfg: cfg,
		ch:  make(chan Alert, 100),
	}
}

// Start launches the background delivery goroutine. Call once at server startup.
func (q *Queue) Start(ctx context.Context) {
	go func() {
		slog.Info("alerts queue started")
		for {
			select {
			case <-ctx.Done():
				slog.Info("alerts queue shutting down")
				return
			case a := <-q.ch:
				q.deliver(a)
			}
		}
	}()
}

// Enqueue adds an alert to the delivery queue.
// Non-blocking: if the buffer is full (100 pending), the alert is dropped with a log warning.
func (q *Queue) Enqueue(a Alert) {
	select {
	case q.ch <- a:
	default:
		slog.Warn("alerts queue full, dropping alert", "channel", a.Channel, "recipient", a.RecipientEmail)
	}
}

// deliver dispatches a single alert via the appropriate channel.
func (q *Queue) deliver(a Alert) {
	var err error
	switch a.Channel {
	case ChannelZohoCliq:
		err = SendZohoCliq(q.cfg, a.Message)
	case ChannelEmail:
		err = SendEmail(q.cfg.SMTPAddr, q.cfg.SMTPUsername, q.cfg.SMTPPassword, q.cfg.SMTPFrom, a.RecipientEmail, a.Subject, a.Message)
	default:
		slog.Warn("alerts unknown channel, falling back to zoho_cliq", "channel", a.Channel)
		err = SendZohoCliq(q.cfg, a.Message)
	}
	if err != nil {
		slog.Error("alerts delivery error", "channel", a.Channel, "recipient", a.RecipientEmail, "error", err)
	}
}
