package grpcapi

import (
	"github.com/manaraph/stream-aggregator/pkg/events"
)

type Dispatcher interface {
	Publish(events.Message)
}
