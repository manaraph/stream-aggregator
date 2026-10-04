package storage_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/manaraph/stream-aggregator/internal/domain"
	"github.com/manaraph/stream-aggregator/internal/storage"
	"github.com/stretchr/testify/require"
)

func TestPostgresBatchInsertAndDuplicateDelivery(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set; skipping PostgreSQL integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	store, err := storage.OpenPostgres(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(store.Close)

	// Opening a second store checks that schema creation is safe on repeated starts.
	secondStore, err := storage.OpenPostgres(ctx, databaseURL)
	require.NoError(t, err)
	secondStore.Close()

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	prefix := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	pattern := prefix + "-%"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = pool.Exec(cleanupCtx, `DELETE FROM sensor_readings WHERE event_id LIKE $1`, pattern)
	})

	eventTime := time.Now().UTC().Truncate(time.Microsecond)
	readings := []domain.Sensor{
		{
			EventID: prefix + "-1", Sensor: "integration-sensor-1",
			MeasurementType: "temperature", Unit: "C", Value: 21.5, Timestamp: eventTime,
		},
		{
			EventID: prefix + "-2", Sensor: "integration-sensor-2",
			MeasurementType: "humidity", Unit: "%", Value: 48, Timestamp: eventTime.Add(time.Second),
		},
	}

	require.NoError(t, store.InsertBatch(ctx, readings))
	// Simulate broker redelivery after the database committed but before its ACK arrived.
	duplicates := append([]domain.Sensor(nil), readings...)
	duplicates[0].Value = -999
	require.NoError(t, store.InsertBatch(ctx, duplicates))

	rows, err := pool.Query(ctx, `
		SELECT event_id, sensor_id, measurement_type, unit, value, event_time
		FROM sensor_readings
		WHERE event_id LIKE $1
		ORDER BY event_id`, pattern)
	require.NoError(t, err)
	defer rows.Close()

	var actual []domain.Sensor
	for rows.Next() {
		var reading domain.Sensor
		require.NoError(t, rows.Scan(
			&reading.EventID, &reading.Sensor, &reading.MeasurementType,
			&reading.Unit, &reading.Value, &reading.Timestamp,
		))
		actual = append(actual, reading)
	}
	require.NoError(t, rows.Err())
	require.Len(t, actual, 2, "duplicate event IDs must not create extra rows")
	require.Equal(t, readings[0].Value, actual[0].Value, "duplicate delivery must not overwrite the committed reading")
	require.Equal(t, readings[0].MeasurementType, actual[0].MeasurementType)
	require.Equal(t, readings[0].Unit, actual[0].Unit)
	require.True(t, readings[0].Timestamp.Equal(actual[0].Timestamp))
	require.Equal(t, readings[1].MeasurementType, actual[1].MeasurementType)
	require.Equal(t, readings[1].Unit, actual[1].Unit)
	require.True(t, readings[1].Timestamp.Equal(actual[1].Timestamp))
}
