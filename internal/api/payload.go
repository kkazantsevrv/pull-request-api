package api

import (
	"encoding/json"
	"net/http"

	"pull-request-api.com/internal/models"
)

func sendError(w http.ResponseWriter, status int, code models.ErrorResponseErrorCode, msg string) {
	errResp := models.ErrorResponse{
		Error: struct {
			Code    models.ErrorResponseErrorCode `json:"code"`
			Message string                        `json:"message"`
		}{
			Code:    code,
			Message: msg,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errResp)
}

// Вспомогательная функция для отправки JSON
func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
