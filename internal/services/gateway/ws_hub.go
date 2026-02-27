package gateway

import streamv1 "github.com/manaraph/stream-aggregator/pkg/pb/stream/v1"

var clients = make(map[*Client]bool)
var broadcast = make(chan *streamv1.SensorEventRequest)
