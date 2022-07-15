package limiter

import (
	"net/http"

	"github.com/go-redis/redis_rate/v9"
	"github.com/marioarizaj/payment-gateway/kit/ctx"
	"github.com/marioarizaj/payment-gateway/kit/responses"
)

func Middleware(l *redis_rate.Limiter, rate int) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			merchantID, err := ctx.GetMerchantID(r.Context())
			if err != nil {
				responses.AuthenticationError(w)
				return
			}
			_, err = l.Allow(r.Context(), merchantID.String(), redis_rate.PerMinute(rate))
			if err != nil {
				responses.TooManyRequestsError(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
