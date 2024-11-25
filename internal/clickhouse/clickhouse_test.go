package clickhouse

import (
	"context"
	"ndiff/config"
	"ndiff/tracker"
	"testing"
)

func new() *DB {
	tracker := tracker.New()
	config := config.New("test")
	return New("test", config.Old, tracker)
}

func TestNew(t *testing.T) {
	db := new()
	defer db.Close()

	db.client.Exec(context.Background(), "SELECT 1")
}
