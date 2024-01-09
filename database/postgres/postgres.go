package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/evocert/lnksnk/database"

	//helper registration posgres server pgx driver
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	//_ "github.com/lib/pq"
)

/*func OpenPGX(ctx context.Context, datasource string) (pgx *PGX, err error) {
	pool, err := pgxpool.New(ctx, datasource)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	if err = pool.Ping(ctx); err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	return &PGX{pool: pool}, nil
}*/

// Open -wrap sql.Open("pgx", datasource)

func Open(datasource string) (db *sql.DB, err error) {
	db, err = sql.Open("pgx/v5", datasource)
	return
}

func OpenPool(datasource string) (db *sql.DB, err error) {
	pxcnfg, pxerr := pgxpool.ParseConfig(datasource)
	if pxerr != nil {
		err = pxerr
		return
	}
	ctx := context.Background()

	pool, err := pgxpool.NewWithConfig(ctx, pxcnfg)
	if err != nil {
		return nil, errors.Wrap(err, "create db conn pool")
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.Wrap(err, "create db conn pool")
	}
	db = stdlib.OpenDBFromPool(pool)
	//db, err = sql.Open("pgx/v5", datasource)
	return
}

func parseSqlParam(totalArgs int) (s string) {
	return "$" + fmt.Sprintf("%d", totalArgs+1)
}

func init() {
	database.GLOBALDBMS().RegisterDriver("postgres", func(datasource string, a ...interface{}) (db *sql.DB, err error) {
		if db, err = OpenPool(datasource); err == nil && db != nil {
			//db.SetMaxOpenConns(1000)
		}
		return
	}, parseSqlParam)
}
