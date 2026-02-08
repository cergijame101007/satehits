package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cergijame101007/satehits/internal/handler"
	"github.com/cergijame101007/satehits/internal/repository"
)

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
	defer db.Close()

	// 実際に接続確認 (Ping)
	if err := db.Ping(); err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	log.Println("Connected to Database!")

	// DI: Repository -> Handler
	repo := repository.NewPostgresReservationRepository(db)
	reservationHandler := handler.NewReservationHandler(repo)

	// ルーティング
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/reservations", reservationHandler.HandleReservations)

	// サーバー起動
	port := ":8080"
	log.Printf("Server starting on %s", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"Welcome to the Go API","status":"success"}`))
}
