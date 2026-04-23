package alerts

import (
	"context"
	"log"
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
		log.Println("alerts: queue started")
		for {
			select {
			case <-ctx.Done():
				log.Println("alerts: queue shutting down")
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
		log.Printf("alerts: queue full — dropping %q alert to %s", a.Channel, a.RecipientEmail)
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
		log.Printf("alerts: unknown channel %q — falling back to zoho_cliq", a.Channel)
		err = SendZohoCliq(q.cfg, a.Message)
	}
	if err != nil {
		log.Printf("alerts: delivery error (channel=%s recipient=%s): %v", a.Channel, a.RecipientEmail, err)
	}
}
