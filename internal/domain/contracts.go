package domain

import (
	"context"
	"time"
)

type EventSink interface {
	Send(ctx context.Context, e Event) error
}

type EventSource interface {
	FetchEvents(ctx context.Context, since time.Time) ([]Event, error)
}
