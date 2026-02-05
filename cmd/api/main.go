package main

import (
	"log"
	"net/http"
	"encoding/json"
)

type JsonResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

type ReservationRequest struct {
	Name string `json:"name"`
	People int `json:"people"`
}

func main() {
	// ルートパスへのハンドラを登録
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/reservations", handleReservations)

	// サーバー起動
	addr := ":8080"
	log.Printf("Server starting on %s", addr)
	
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	// GETメソッド以外は405 Method Not Allowedを返す
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Root以外のときは404 Not Foundを返す
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	message := JsonResponse{
		Message: "Welcome to the Go API",
		Status: "success",
	}

	respondWithJSON(w, http.StatusOK, message)
}

func handleReservations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.URL.Path != "/reservations" {
		http.NotFound(w, r)
		return
	}

	var request ReservationRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received Reservation: Name=%s, People=%d", request.Name, request.People)

	responseMessage := JsonResponse{
		Message: "Reservation created",
		Status: "success",
	}

	respondWithJSON(w, http.StatusOK, responseMessage)
}

func respondWithJSON(w http.ResponseWriter, status int, payload interface{}) {
	res, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(res)
}
