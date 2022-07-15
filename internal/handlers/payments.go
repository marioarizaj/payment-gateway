package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/marioarizaj/payment-gateway"
	ctx2 "github.com/marioarizaj/payment-gateway/kit/ctx"
	"github.com/marioarizaj/payment-gateway/kit/responses"
)

func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var payment payment_gateway.Payment
	err := json.NewDecoder(r.Body).Decode(&payment)
	if err != nil {
		log.Printf("could not decode request body: %v", err)
		responses.RespondWithError(w, http.StatusBadRequest, "could not decode request body")
		return
	}
	ctx := r.Context()

	merchantID, err := ctx2.GetMerchantID(ctx)
	if err != nil {
		responses.AuthenticationError(w)
		return
	}
	payment.MerchantID = merchantID

	_, err = h.domain.GetPayment(ctx, payment.ID)
	if err == nil {
		// This means that this payment was created some time in the past
		responses.RespondWithJSON(w, http.StatusCreated, payment)
		return
	}
	var notFoundErr responses.NotFoundError
	if !errors.As(err, &notFoundErr) {
		var resErr responses.ResponseError
		if errors.As(err, &resErr) {
			resErr.Response(w)
			return
		}
	}

	payment, err = h.domain.CreatePayment(ctx, payment)
	if err != nil {
		var resErr responses.ResponseError
		if errors.As(err, &resErr) {
			resErr.Response(w)
			return
		}
	}
	responses.RespondWithJSON(w, http.StatusCreated, payment)
}

func (h *Handler) GetPayment(w http.ResponseWriter, r *http.Request) {
	id, exists := mux.Vars(r)["id"]
	if !exists {
		responses.RespondWithError(w, http.StatusBadRequest, "id not found in request")
		return
	}
	ctx := r.Context()
	uid, err := uuid.Parse(id)
	if err != nil {
		responses.RespondWithError(w, http.StatusBadRequest, "id format not accurate")
		return
	}
	payment, err := h.domain.GetPayment(ctx, uid)
	if err != nil {
		var resErr responses.ResponseError
		if errors.As(err, &resErr) {
			resErr.Response(w)
			return
		}
	}
	responses.RespondWithJSON(w, http.StatusOK, payment)
}
