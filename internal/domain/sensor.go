package domain

import "time"

type Sensor struct {
	Sensor    string    `json:"sensor"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}
