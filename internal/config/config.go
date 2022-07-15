package config

import (
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

const (
	devEnv      = "dev"
	testEnv     = "test"
	projectName = "payment-gateway"
)

type DatabaseConfig struct {
	DatabaseURL string `envconfig:"DATABASE_URL"`
}

type AppConfig struct {
	AppEnv string `envconfig:"APP_ENV"`
}

type Server struct {
	Port         int   `envconfig:"SERVER_PORT"`
	IdleTimeout  int64 `envconfig:"SERVER_IDLE_TIMEOUT"`
	ReadTimeout  int64 `envconfig:"SERVER_READ_TIMEOUT"`
	WriteTimeout int64 `envconfig:"SERVER_WRITE_TIMEOUT"`
}

type Auth struct {
	ApiKeySecret string `envconfig:"API_KEY_SECRET"`
}

type Redis struct {
	Addr     string `envconfig:"REDIS_ADDRESS"`
	Password string `envconfig:"REDIS_PASSWORD"`
	DB       int    `envconfig:"REDIS_DB"`
}

type RateLimiter struct {
	AllowedReqsPerSecond int `envconfig:"ALLOWED_REQUESTS_PER_SECOND"`
}

type CircuitBreakerConfig struct {
	Commands               []string `envconfig:"HYSTRIX_COMMANDS"`
	Timeout                int      `envconfig:"HYSTRIX_TIMEOUT"`
	MaxConcurrentRequests  int      `envconfig:"HYSTRIX_MAX_CONCURRENT_REQUESTS"`
	ErrorPercentThreshold  int      `envconfig:"HYSTRIX_ERROR_PERCENT_THRESHOLD"`
	RequestVolumeThreshold int      `envconfig:"HYSTRIX_REQUEST_VOLUME_THRESHOLD"`
	SleepWindow            int      `envconfig:"HYSTRIX_SLEEP_WINDOW"`
}

type MockBankConfig struct {
	StatusCode                  int    `envconfig:"MOCK_STATUS_CODE" default:"202"`
	UpdateToStatus              string `envconfig:"MOCK_PAYMENT_STATUS" default:"succeeded"`
	SleepIntervalInitialRequest int    `envconfig:"SLEEP_INTERVAL_INITIAL_REQUEST" default:"10"`
	SleepIntervalForCallback    int    `envconfig:"SLEEP_INTERVAL_FOR_CALLBACK" default:"200"`
	ShouldRunCallback           bool   `envconfig:"SHOULD_RUN_CALLBACK" default:"true"`
	FailedReason                string `envconfig:"MOCK_FAILED_REASON"`
}

type Config struct {
	AppConfig            AppConfig
	Server               Server
	RateLimiter          RateLimiter
	Auth                 Auth
	Redis                Redis
	MockBankConfig       MockBankConfig
	CircuitBreakerConfig CircuitBreakerConfig
	DatabaseConfig       DatabaseConfig
}

func LoadConfig() (Config, error) {
	appEnv := getAppEnv()
	var envPath []string
	if appEnv == testEnv || appEnv == devEnv {
		// To run this locally, we need to open .env
		err := filepath.Walk(os.ExpandEnv("$GOPATH/src"), func(path string, info fs.FileInfo, err error) error {
			if strings.Contains(path, projectName) && strings.Contains(path, ".env") && !strings.Contains(path, "docker") {
				envPath = append(envPath, path)
			}
			return nil
		})
		if err != nil {
			log.Println("could not open .env file, skipping")
		}
		err = godotenv.Load(envPath...)
		if err != nil {
			log.Println("could not open .env file, skipping")
		}
	}
	var config Config
	err := envconfig.Process("", &config)
	return config, err
}

func RootDir() string {
	_, b, _, _ := runtime.Caller(1)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}

func getAppEnv() string {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "" {
		return devEnv
	}
	return appEnv
}
