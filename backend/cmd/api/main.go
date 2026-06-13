package main

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	authusecase "github.com/cergijame101007/satehits/internal/application/usecase/auth"
	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	scheduleusecase "github.com/cergijame101007/satehits/internal/application/usecase/schedule"
	"github.com/cergijame101007/satehits/internal/domain/service"
	"github.com/cergijame101007/satehits/internal/handler"
	"github.com/cergijame101007/satehits/internal/repository"
	"github.com/cergijame101007/satehits/pkg/config"
	"github.com/cergijame101007/satehits/pkg/jwt"
)

const (
	apiVersion = "v1"
	adminBase  = "/api/" + apiVersion + "/admin"
)

func main() {
	// .envファイルを読み込む
	if err := godotenv.Load(); err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	cfg := config.Load()

	// DB接続を開く
	db, err := sql.Open("pgx", cfg.DatabaseURL)
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
	reservationsPath := "/api/" + apiVersion + "/reservations"
	schedulesPath := adminBase + "/schedules"

	reservationRepo := repository.NewPostgresReservationRepository(db)
	createReservation := reservationusecase.NewCreateReservationUseCase(reservationRepo)
	reservationHandler := handler.NewReservationHandler(reservationRepo, createReservation, reservationsPath)

	scheduleRepo := repository.NewPostgresScheduleRepository(db)
	scheduleResolver := service.NewScheduleResolver(scheduleRepo)
	setSchedule := scheduleusecase.NewSetScheduleUseCase(scheduleRepo)
	listSchedules := scheduleusecase.NewListSchedulesUseCase(scheduleResolver)
	getSchedule := scheduleusecase.NewGetScheduleUseCase(scheduleResolver)
	scheduleHandler := handler.NewScheduleHandler(setSchedule, listSchedules, getSchedule, schedulesPath)

	// 認証（AT/RT）の DI
	jwtService := jwt.NewJWTService(cfg.JWTSecret, "satehits-api", "satehits-admin", time.Hour)
	adminUserRepo := repository.NewPostgresAdminUserRepository(db)
	refreshTokenRepo := repository.NewPostgresRefreshTokenRepository(db)
	txManager := repository.NewTxManager(db)

	loginUC := authusecase.NewLoginUseCase(adminUserRepo, refreshTokenRepo, jwtService)
	refreshUC := authusecase.NewRefreshUseCase(adminUserRepo, refreshTokenRepo, jwtService, txManager)
	logoutUC := authusecase.NewLogoutUseCase(refreshTokenRepo)
	authHandler := handler.NewAuthHandler(loginUC, refreshUC, logoutUC, cfg.CORSOrigins, cfg.CookieDomain)

	// ルーティング（公開 API は /api/v1/...）
	http.HandleFunc("/", handleRoot)
	http.HandleFunc(reservationsPath, reservationHandler.HandleReservations)

	// 認証エンドポイント、login / refresh は AT 不要
	http.HandleFunc(adminBase+"/login", authHandler.HandleLogin)
	http.HandleFunc(adminBase+"/refresh", authHandler.HandleRefresh)
	// logout は有効な AT 必須 → RequireAuth でラップ
	http.Handle(adminBase+"/logout", handler.RequireAuth(jwtService)(http.HandlerFunc(authHandler.HandleLogout)))

	scheduleAuth := handler.RequireAuth(jwtService)
	http.Handle(schedulesPath, scheduleAuth(http.HandlerFunc(scheduleHandler.HandleSchedules)))
	http.Handle(schedulesPath+"/", scheduleAuth(http.HandlerFunc(scheduleHandler.HandleSchedules)))

	// サーバー起動
	port := ":8080"
	log.Printf("Server starting on %s", port)

	srv := &http.Server{
		Addr:         port,
		Handler:      handler.CORS(cfg.CORSOrigins)(http.DefaultServeMux),
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
