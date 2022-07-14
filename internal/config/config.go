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
	projectName = "payment_gateway"
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

type Config struct {
	AppConfig      AppConfig
	Server         Server
	RateLimiter    RateLimiter
	Auth           Auth
	Redis          Redis
	DatabaseConfig DatabaseConfig
}

func LoadConfig() (Config, error) {
	appEnv := getAppEnv()
	envPath := ""
	if appEnv == testEnv || appEnv == devEnv {
		err := filepath.Walk(os.ExpandEnv("$GOPATH/src"), func(path string, info fs.FileInfo, err error) error {
			if strings.Contains(path, projectName) && strings.Contains(path, ".env") {
				envPath = path
			}
			return nil
		})
		if err != nil {
			log.Println("could not open .env file, skipping")
		}
		err = godotenv.Load(envPath)
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
