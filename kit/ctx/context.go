package ctx

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

type ctxKey int

const merchantIdKey ctxKey = 1

var notFoundError = errors.New("not found")

func AddMerchantID(ctx context.Context, value uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, merchantIdKey, value)
	return ctx
}

func GetMerchantID(ctx context.Context) (uuid.UUID, error) {
	value, ok := ctx.Value(merchantIdKey).(uuid.UUID)
	if !ok {
		return uuid.UUID{}, notFoundError
	}
	return value, nil
}
