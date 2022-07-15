package dependencies

import (
	"context"
	"database/sql"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/marioarizaj/payment_gateway/internal/acquiringbank"
	"time"

	"github.com/go-redis/redis_rate/v9"

	"github.com/go-redis/redis/v8"
	"github.com/marioarizaj/payment_gateway/internal/config"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

type Dependencies struct {
	DB         bun.IDB
	Limiter    *redis_rate.Limiter
	BankClient *acquiringbank.MockClient
	Redis      *redis.Client
}

func InitDependencies(config config.Config) (Dependencies, error) {
	ConfigureHystrix(config.CircuitBreakerConfig)
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
		// By default, let's always return a good response
		BankClient: &acquiringbank.MockClient{
			StatusCode:                  202,
			SleepIntervalInitialRequest: 20 * time.Millisecond,
			SleepIntervalForCallback:    60 * time.Millisecond,
			ShouldRunCallback:           true,
		},
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

// ConfigureHystrix sets up hystrix circuit breakers.
func ConfigureHystrix(cfg config.CircuitBreakerConfig) {
	for _, c := range cfg.Commands {
		hystrix.ConfigureCommand(c, hystrix.CommandConfig{
			Timeout:                cfg.Timeout,
			MaxConcurrentRequests:  cfg.MaxConcurrentRequests,
			ErrorPercentThreshold:  cfg.ErrorPercentThreshold,
			RequestVolumeThreshold: cfg.RequestVolumeThreshold,
			SleepWindow:            cfg.SleepWindow,
		})
	}
}
