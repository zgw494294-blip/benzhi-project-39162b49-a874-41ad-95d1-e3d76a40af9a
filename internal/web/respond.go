package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"benzhi-project-39162b49-a874-41ad-95d1-e3d76a40af9a/internal/domain"
)

type errorEnvelope struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code    string              `json:"code"`
	Message string              `json:"message"`
	Fields  []domain.FieldError `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string, fields []domain.FieldError) {
	writeJSON(w, status, errorEnvelope{Error: apiError{Code: code, Message: message, Fields: fields}})
}

func handleServiceError(w http.ResponseWriter, err error) {
	var validation *domain.ValidationError
	switch {
	case errors.As(err, &validation):
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", validation.Error(), validation.Fields)
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrConflict):
		writeError(w, http.StatusConflict, "VERSION_CONFLICT", "方案已被更新，请刷新后重试", nil)
	case errors.Is(err, domain.ErrRoleSeparation):
		writeError(w, http.StatusUnprocessableEntity, "ROLE_SEPARATION", err.Error(), nil)
	case errors.Is(err, domain.ErrInvalidState), errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusUnprocessableEntity, "INVALID_OPERATION", err.Error(), nil)
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误", nil)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求 JSON 无法解析："+err.Error(), nil)
		return false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "请求只能包含一个 JSON 对象", nil)
		return false
	}
	return true
}

func requestKey(r *http.Request, bodyValue string) string {
	if value := r.Header.Get("Idempotency-Key"); value != "" {
		return value
	}
	return bodyValue
}
