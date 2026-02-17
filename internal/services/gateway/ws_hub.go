package gateway

import pb "github.com/manaraph/stream-aggregator/proto"

var clients = make(map[*Client]bool)
var broadcast = make(chan *pb.SensorData)
