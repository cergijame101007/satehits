package handler

import (
	"encoding/json"
	"net/http"
)

// JSONResponse はAPIレスポンスの共通構造体
type JSONResponse struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// respondWithJSON はJSONレスポンスを返すヘルパー関数
func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	res, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(res); err != nil {
		// Log the error but don't try to send another response as headers are already written
		return
	}
}
