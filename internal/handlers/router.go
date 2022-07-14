package handlers

import (
	"github.com/marioarizaj/payment_gateway/internal/config"
	"github.com/marioarizaj/payment_gateway/kit/auth"
	"github.com/marioarizaj/payment_gateway/kit/limiter"
	"github.com/marioarizaj/payment_gateway/kit/logging"
	"github.com/marioarizaj/payment_gateway/kit/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/marioarizaj/payment_gateway/internal/dependencies"
	"github.com/marioarizaj/payment_gateway/internal/domain/payment"
	"github.com/marioarizaj/payment_gateway/internal/repositiory"
	"github.com/marioarizaj/payment_gateway/kit/rediscache"
)

type Handler struct {
	domain *payment.Domain
}

func NewRouter(cfg config.Config, deps dependencies.Dependencies, l *zap.Logger) *mux.Router {
	h := &Handler{
		domain: payment.NewDomain(repositiory.NewRepository(deps.DB), rediscache.NewRedisClient(deps.Redis), l),
	}

	r := mux.NewRouter()
	r.Handle("/metrics", promhttp.Handler())
	v1R := r.PathPrefix("/v1").Subrouter()
	v1R.Use(auth.Middleware(cfg.Auth.ApiKeySecret))
	v1R.Use(limiter.Middleware(deps.Limiter, cfg.RateLimiter.AllowedReqsPerSecond))
	v1R.Use(prometheus.Middleware)
	v1R.Use(logging.Middleware(l))
	v1R.HandleFunc("/payments", h.CreatePayment).Methods(http.MethodPost)
	v1R.HandleFunc("/payments/{id}", h.GetPayment).Methods(http.MethodGet)

	return r
}
