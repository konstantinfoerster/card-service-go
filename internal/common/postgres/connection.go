package postgres

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/konstantinfoerster/card-service/internal/config"
	"runtime"
	"time"
)

type DBConnection struct {
	Conn   DBConn
	pgxCon *pgxpool.Pool
}

func Connect(ctx context.Context, config config.Database) (*DBConnection, error) {
	c, err := pgxpool.ParseConfig(config.ConnectionUrl())
	if err != nil {
		return nil, err
	}
	c.MaxConnLifetime = time.Second * 5
	c.MaxConnIdleTime = time.Millisecond * 500
	c.HealthCheckPeriod = time.Millisecond * 500
	c.MaxConns = int32(runtime.NumCPU()) + 5 // + 5 just in case

	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	dbConn := &DBConnection{
		Conn:   pool,
		pgxCon: pool,
	}

	return dbConn, err
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
		return pgx.BeginTxFunc(ctx, d.pgxCon, opts, func(t pgx.Tx) error {
			dbCon := &DBConnection{
				Conn:   t,
				pgxCon: d.pgxCon,
			}
			return f(dbCon)
		})
	}
}

// DBConn implemented by pgx.Conn and pgx.Tx
type DBConn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...interface{}) pgx.Row
}
