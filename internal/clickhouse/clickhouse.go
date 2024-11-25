package clickhouse

import (
	"context"
	"database/sql"
	"ndiff"
	"ndiff/config"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const tag = "clickhouse"

type Rows driver.Rows

var ErrNoRows = sql.ErrNoRows

type DB struct {
	name    string
	config  config.Clickhouse
	tracker ndiff.Tracker

	client driver.Conn
}

func New(name string, config config.Clickhouse, tracker ndiff.Tracker) *DB {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{config.Address},
		Auth: clickhouse.Auth{
			Database: config.Database,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:          time.Second * 30,
		MaxOpenConns:         5,
		MaxIdleConns:         5,
		ConnMaxLifetime:      time.Duration(10) * time.Minute,
		ConnOpenStrategy:     clickhouse.ConnOpenInOrder,
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10240,
		ClientInfo: clickhouse.ClientInfo{ // optional, please see Client info section in the README.md
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: name, Version: "0.1"},
			},
		},
	})
	if err != nil {
		tracker.Fatal(tag, "failed to open clickhouse", ndiff.ErrorTag(err), ndiff.NewTag("name", name))
	}
	err = conn.Ping(context.Background())
	if err != nil {
		tracker.Fatal(tag, "failed to ping clickhouse", ndiff.ErrorTag(err), ndiff.NewTag("name", name))
	}

	return &DB{
		name:    name,
		config:  config,
		tracker: tracker,
		client:  conn,
	}
}

func (db *DB) QueryAll(ctx context.Context, query string, args ...any) (Rows, error) {
	return db.client.Query(ctx, query, args...)
}

func (db *DB) Close() {
	db.client.Close()
}

func (db *DB) Name() string {
	return db.name
}
