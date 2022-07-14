package dependencies

import (
	"context"
	"database/sql"

	"github.com/go-redis/redis_rate/v9"

	"github.com/go-redis/redis/v8"
	"github.com/marioarizaj/payment_gateway/internal/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type Dependencies struct {
	DB      bun.IDB
	Limiter *redis_rate.Limiter
	Redis   *redis.Client
}

func InitDependencies(config config.Config) (Dependencies, error) {
	db, err := InitDB(config.DatabaseConfig.DatabaseURL)
	if err != nil {
		return Dependencies{}, err
	}
	rds, err := InitRedis(config.Redis)
	if err != nil {
		return Dependencies{}, err
	}
	return Dependencies{
		DB:      db,
		Limiter: redis_rate.NewLimiter(rds),
		Redis:   rds,
	}, nil
}

func InitDB(dsn string) (bun.IDB, error) {
	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	err := sqldb.Ping()
	bunDB := bun.NewDB(sqldb, pgdialect.New())
	return bunDB, err
}

func InitRedis(cfg config.Redis) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password, // no password set
		DB:       cfg.DB,       // use default DB
	})
	err := rdb.Ping(context.Background()).Err()
	return rdb, err
}
