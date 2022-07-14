package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/marioarizaj/payment_gateway/kit/ctx"
	"github.com/marioarizaj/payment_gateway/kit/responses"
)

func GetHMAC(uuid uuid.UUID, secret string) string {
	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(secret))

	// Write Data to it
	_, err := h.Write([]byte(uuid.String()))
	if err != nil {
		log.Printf("error when writing data to hash: %v", err)
	}
	// Get result and encode as hexadecimal string
	sha := hex.EncodeToString(h.Sum(nil))
	return sha
}

func Middleware(secret string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			username, password, ok := r.BasicAuth()
			if !ok {
				responses.AuthenticationError(w)
				return
			}
			uid, err := uuid.Parse(username)
			if err != nil {
				responses.AuthenticationError(w)
				return
			}
			sha := GetHMAC(uid, secret)
			equal := hmac.Equal([]byte(sha), []byte(password))
			if !equal {
				responses.AuthenticationError(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx.AddMerchantID(r.Context(), uid)))
		})
	}
}
