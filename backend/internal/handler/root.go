package handler

import "net/http"

// rootMessage は GET / の本文。DB の状態は含まない（DB まで確かめるのは GET /healthz。docs/monitoring.md §4.4）
const rootMessage = "satehits API is running"

// HandleRoot は GET / のプロセス応答確認。他のハンドラに一致しないパスもここに来るため 404 JSON を返す
func HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		respondNotFound(w)
		return
	}
	if r.Method != http.MethodGet {
		respondMethodNotAllowed(w, http.MethodGet)
		return
	}

	respondWithJSON(w, http.StatusOK, JSONResponse{Message: rootMessage, Status: healthStatusOK})
}
