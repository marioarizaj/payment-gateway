package responses

import (
	"encoding/json"
	"errors"
	"net/http"
)

type ResponseError interface {
	Error() string
	Response(w http.ResponseWriter)
}

type BadRequestError struct {
	Err error
}

func (e BadRequestError) Error() string {
	return e.Err.Error()
}

func (e BadRequestError) Response(w http.ResponseWriter) {
	RespondWithError(w, http.StatusBadRequest, e.Error())
}

type TooManyRequests struct {
}

func (e TooManyRequests) Error() string {
	return "too many requests"
}

func (e TooManyRequests) Response(w http.ResponseWriter) {
	RespondWithError(w, http.StatusTooManyRequests, e.Error())
}

type InternalServerError struct {
	Err error
}

func (e InternalServerError) Error() string {
	return e.Err.Error()
}

func (e InternalServerError) Response(w http.ResponseWriter) {
	RespondWithError(w, http.StatusInternalServerError, e.Error())
}

type NotFoundError struct{}

func (e NotFoundError) Error() string {
	return "not found"
}

func (e NotFoundError) Response(w http.ResponseWriter) {
	RespondWithError(w, http.StatusNotFound, e.Error())
}

type ConflictError struct{}

func (e ConflictError) Error() string {
	return "conflict"
}

func (e ConflictError) Response(w http.ResponseWriter) {
	RespondWithError(w, http.StatusConflict, e.Error())
}

func AuthenticationError(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Basic realm=<realm>")
	RespondWithError(w, http.StatusUnauthorized, "unauthorized")
}

func RespondWithError(w http.ResponseWriter, code int, message string) {
	RespondWithJSON(w, code, map[string]string{"error": message})
}

func TooManyRequestsError(w http.ResponseWriter) {
	RespondWithError(w, http.StatusTooManyRequests, "too many requests")
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write(response)
}

func GetErrorResponseFromStatusCode(code int) error {
	switch code {
	case 500:
		return InternalServerError{Err: errors.New("internal server error")}
	case 400:
		return BadRequestError{Err: errors.New("bad request")}
	case 429:
		return TooManyRequests{}
	case 409:
		return ConflictError{}
	default:
		return InternalServerError{}
	}
}
