package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQLドライバ
)

type JsonResponse struct {
	Message string `json:"message"`
	Status string `json:"status"`
}

type ReservationRequest struct {
	Name string `json:"name"`
	People int `json:"people"`
}

// ReservationRepositoryは、予約データを永続化するためのリポジトリ
type ReservationRepository struct {
	db *sql.DB
}

// NewReservationRepositoryは、ReservationRepositoryのインスタンスを作成する
func NewReservationRepository(db *sql.DB) *ReservationRepository {
	return &ReservationRepository{db: db}
}

// Createは、予約データを永続化する
func (r *ReservationRepository) Create(ctx context.Context, name string, people int) error {
	query := `INSERT INTO reservations (name, people) VALUES ($1, $2)`
	_, err := r.db.ExecContext(ctx, query, name, people)
	return err
}

// Serverは、APIサーバーを管理する
type Server struct {
	repo *ReservationRepository
}

func main() {
	// .envファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// DBのURLを取得
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	// DB接続を開く
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("Unable to parse DB URL: %v", err)
	}
	// main終了時に閉じる
	defer db.Close()

	// 実際に接続確認 (Ping)
	if err := db.Ping(); err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	log.Println("✅ Connected to Database!")

	// ReservationRepositoryのインスタンスを作成
	repo := NewReservationRepository(db)
	// Serverのインスタンスを作成
	server := &Server{repo: repo}

	// ルートパスへのハンドラを登録
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/reservations", server.handleReservations)

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

func (s *Server) handleReservations(w http.ResponseWriter, r *http.Request) {
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

	// 簡易バリデーション
	if request.Name == "" || request.People <= 0 {
		http.Error(w, "Name and valid people count required", http.StatusBadRequest)
		return
	}

	// Repositoryを使って予約データを永続化
	if err := s.repo.Create(r.Context(), request.Name, request.People); err != nil {
		log.Printf("Failed to create reservation: %v", err)
		http.Error(w, "Failed to create reservation", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Saved Reservation: Name=%s, People=%d", request.Name, request.People)

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
