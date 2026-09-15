package handler

import "net/http"

// HandleRoot は GET / のヘルスチェック。他のハンドラに一致しないパスもここに来るため 404 JSON を返す
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		respondNotFound(w)
		return
	}
	if r.Method != http.MethodGet {
		respondMethodNotAllowed(w, http.MethodGet)
		return
	}

	respondWithJSON(w, http.StatusOK, JSONResponse{Message: "Welcome to the Go API", Status: "success"})
}
