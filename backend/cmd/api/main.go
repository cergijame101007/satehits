package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	scheduleusecase "github.com/cergijame101007/satehits/internal/application/usecase/schedule"
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

	// DI: Repository -> UseCase -> Handler
	const apiVersion = "v1"

	reservationRepo := repository.NewPostgresReservationRepository(db)
	createReservation := reservationusecase.NewCreateReservationUseCase(reservationRepo)
	reservationsPath := fmt.Sprintf("/api/%s/reservations", apiVersion)
	reservationHandler := handler.NewReservationHandler(reservationRepo, createReservation, reservationsPath)

	scheduleRepo := repository.NewPostgresScheduleRepository(db)
	createSchedule := scheduleusecase.NewCreateScheduleUseCase(scheduleRepo)
	schedulesPath := fmt.Sprintf("/api/%s/admin/schedules", apiVersion)
	scheduleHandler := handler.NewScheduleHandler(scheduleRepo, createSchedule, schedulesPath)

	// ルーティング（公開 API は /api/v1/...）
	http.HandleFunc("/", handleRoot)
	http.HandleFunc(reservationsPath, reservationHandler.HandleReservations)
	// TODO: スケジュール管理は管理者 JWT 認証必須、認証ミドルウェア実装後に
	// 本ルートをラップしてからハンドラへ渡す（現状は開発用に無認証）
	http.HandleFunc(schedulesPath, scheduleHandler.HandleSchedules)

	// サーバー起動
	port := ":8080"
	log.Printf("Server starting on %s", port)

	srv := &http.Server{
		Addr:         port,
		Handler:      nil,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
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
	if _, err := w.Write([]byte(`{"message":"Welcome to the Go API","status":"success"}`)); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}
