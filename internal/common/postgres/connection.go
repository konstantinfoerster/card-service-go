package postgres

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/konstantinfoerster/card-service-go/internal/common/config"
)

type DBConnection struct {
	Conn   DBConn
	pgxCon *pgxpool.Pool
}

func Connect(ctx context.Context, config config.Database) (*DBConnection, error) {
	c, err := pgxpool.ParseConfig(config.ConnectionURL())
	if err != nil {
		return nil, fmt.Errorf("failed to parse config from URL %w", err)
	}
	c.MaxConnLifetime = time.Second * time.Duration(5)
	c.MaxConnIdleTime = time.Millisecond * time.Duration(500)
	c.HealthCheckPeriod = time.Millisecond * time.Duration(500)
	var maxConnBuffer int32 = 5 // + 5 just in case
	c.MaxConns = int32(runtime.NumCPU()) + maxConnBuffer

	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool %w", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping database %w", err)
	}

	dbConn := &DBConnection{
		Conn:   pool,
		pgxCon: pool,
	}

	return dbConn, nil
}

func (d *DBConnection) Close() error {
	d.pgxCon.Close()

	return nil
}

func (d *DBConnection) WithTransaction(ctx context.Context, f func(conn *DBConnection) error) error {
	switch d.Conn.(type) {
	case pgx.Tx:
		return fmt.Errorf("already inside a transaction")
	default:
		opts := pgx.TxOptions{AccessMode: pgx.ReadWrite, IsoLevel: pgx.ReadCommitted}

		if err := pgx.BeginTxFunc(ctx, d.pgxCon, opts, func(t pgx.Tx) error {
			return f(&DBConnection{
				Conn:   t,
				pgxCon: d.pgxCon,
			})
		}); err != nil {
			return fmt.Errorf("transaction error %w", err)
		}

		return nil
	}
}

// DBConn implemented by pgx.Conn and pgx.Tx.
type DBConn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}
