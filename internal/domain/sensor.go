package domain

import "time"

type Sensor struct {
	EventID         string    `json:"eventId,omitempty"`
	Sensor          string    `json:"sensor"`
	MeasurementType string    `json:"measurementType"`
	Unit            string    `json:"unit"`
	Value           float64   `json:"value"`
	Timestamp       time.Time `json:"timestamp"`
}
