package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	authusecase "github.com/cergijame101007/satehits/internal/application/usecase/auth"
	reservationusecase "github.com/cergijame101007/satehits/internal/application/usecase/reservation"
	scheduleusecase "github.com/cergijame101007/satehits/internal/application/usecase/schedule"
	supplierusecase "github.com/cergijame101007/satehits/internal/application/usecase/supplier"
	"github.com/cergijame101007/satehits/internal/domain"
	"github.com/cergijame101007/satehits/internal/domain/service"
	"github.com/cergijame101007/satehits/internal/handler"
	"github.com/cergijame101007/satehits/internal/infrastructure/external/resend"
	"github.com/cergijame101007/satehits/internal/infrastructure/external/storage"
	"github.com/cergijame101007/satehits/internal/infrastructure/external/turnstile"
	inframail "github.com/cergijame101007/satehits/internal/infrastructure/mail"
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
	scheduleRepo := repository.NewPostgresScheduleRepository(db)
	scheduleResolver := service.NewScheduleResolver(scheduleRepo)
	availabilityService := service.NewAvailabilityService(scheduleResolver, reservationRepo)

	getAvailability := reservationusecase.NewGetAvailabilityUseCase(availabilityService)
	availabilityPath := reservationsPath + "/availability"
	availabilityHandler := handler.NewAvailabilityHandler(getAvailability, availabilityPath)

	publicSchedulesPath := "/api/" + apiVersion + "/schedules"
	publicScheduleHandler := handler.NewPublicScheduleHandler(getAvailability, publicSchedulesPath)

	listReservations := reservationusecase.NewListReservationsUseCase(reservationRepo)
	createAdminReservation := reservationusecase.NewCreateAdminReservationUseCase(reservationRepo)

	txManager := repository.NewTxManager(db)

	var mailSender domain.MailSender = resend.NoOpSender{}
	if cfg.ResendAPIKey != "" {
		mailSender = resend.NewClient(cfg.ResendAPIKey, nil)
		log.Println("Resend mail sender enabled")
	} else {
		log.Println("RESEND_API_KEY not set; mail sender runs in noop mode")
	}
	emailOutboxRepo := repository.NewPostgresEmailOutboxRepository(db)
	mailEnqueuer := inframail.NewOutboxEnqueuer(emailOutboxRepo, cfg.MailFromAddress)
	mailDispatcher := inframail.NewDispatcher(emailOutboxRepo, mailSender, inframail.Config{
		BatchSize:  cfg.OutboxBatchSize,
		TimeBudget: time.Duration(cfg.OutboxFlushTimeBudgetSeconds) * time.Second,
	})

	updateReservationStatus := reservationusecase.NewUpdateReservationStatusUseCase(reservationRepo, txManager, mailEnqueuer)
	adminReservationsPath := adminBase + "/reservations"
	adminReservationHandler := handler.NewAdminReservationHandler(
		listReservations,
		createAdminReservation,
		updateReservationStatus,
		adminReservationsPath,
	)

	setSchedule := scheduleusecase.NewSetScheduleUseCase(scheduleRepo)
	listSchedules := scheduleusecase.NewListSchedulesUseCase(scheduleResolver)
	getSchedule := scheduleusecase.NewGetScheduleUseCase(scheduleResolver)
	deleteSchedule := scheduleusecase.NewDeleteScheduleUseCase(scheduleRepo)
	scheduleHandler := handler.NewScheduleHandler(setSchedule, listSchedules, getSchedule, deleteSchedule, schedulesPath)

	// 取引先（Supplier）の DI
	supplierRepo := repository.NewPostgresSupplierRepository(db)
	supplierTxManager := repository.NewTxManager(db)

	var imageStorage domain.ImageStorage = storage.NoOpStorage{}
	if cfg.Storage.Enabled {
		imageStorage = storage.NewClient(storage.Config{
			Endpoint:      cfg.Storage.Endpoint,
			Region:        cfg.Storage.Region,
			Bucket:        cfg.Storage.Bucket,
			AccessKey:     cfg.Storage.AccessKey,
			SecretKey:     cfg.Storage.SecretKey,
			PublicBaseURL: cfg.Storage.PublicBaseURL,
		})
		log.Println("Image storage enabled")
	} else {
		log.Println("STORAGE_* not fully set; image upload is disabled")
	}

	listSuppliers := supplierusecase.NewListSuppliersUseCase(supplierRepo)
	getSupplier := supplierusecase.NewGetSupplierUseCase(supplierRepo)
	createSupplier := supplierusecase.NewCreateSupplierUseCase(supplierRepo)
	updateSupplier := supplierusecase.NewUpdateSupplierUseCase(supplierRepo)
	deleteSupplier := supplierusecase.NewDeleteSupplierUseCase(supplierRepo)
	reorderSuppliers := supplierusecase.NewReorderSuppliersUseCase(supplierRepo, supplierTxManager)
	uploadSupplierImage := supplierusecase.NewUploadImageUseCase(supplierRepo, imageStorage)

	publicSuppliersPath := "/api/" + apiVersion + "/suppliers"
	publicSupplierHandler := handler.NewPublicSupplierHandler(listSuppliers, publicSuppliersPath)

	adminSuppliersPath := adminBase + "/suppliers"
	adminSupplierHandler := handler.NewAdminSupplierHandler(
		listSuppliers,
		getSupplier,
		createSupplier,
		updateSupplier,
		deleteSupplier,
		reorderSuppliers,
		uploadSupplierImage,
		adminSuppliersPath,
	)

	// 認証（AT/RT）の DI
	jwtService := jwt.NewJWTService(cfg.JWTSecret, "satehits-api", "satehits-admin", time.Hour)
	adminUserRepo := repository.NewPostgresAdminUserRepository(db)
	refreshTokenRepo := repository.NewPostgresRefreshTokenRepository(db)

	var captchaVerifier reservationusecase.CaptchaVerifier = reservationusecase.NoOpCaptchaVerifier{}
	if cfg.Environment != "development" {
		captchaVerifier = turnstile.NewVerifier(cfg.TurnstileSecret, nil)
	}

	visitDateLocker := repository.PostgresVisitDateLocker{}
	createReservation := reservationusecase.NewCreateReservationUseCase(
		reservationRepo, scheduleResolver, availabilityService, txManager,
		visitDateLocker, captchaVerifier, mailEnqueuer,
	)
	reservationHandler := handler.NewReservationHandler(createReservation, reservationsPath)

	loginAttemptRepo := repository.NewPostgresLoginAttemptRepository(db)
	loginUC := authusecase.NewLoginUseCase(
		adminUserRepo,
		refreshTokenRepo,
		loginAttemptRepo,
		jwtService,
		authusecase.LoginRateLimitPolicy{
			EmailMax: cfg.LoginRateLimit.EmailMax,
			IPMax:    cfg.LoginRateLimit.IPMax,
			Window:   time.Duration(cfg.LoginRateLimit.WindowMinutes) * time.Minute,
		},
	)
	refreshUC := authusecase.NewRefreshUseCase(adminUserRepo, refreshTokenRepo, jwtService, txManager)
	logoutUC := authusecase.NewLogoutUseCase(refreshTokenRepo)
	authHandler := handler.NewAuthHandler(
		loginUC, refreshUC, logoutUC, cfg.CORSOrigins, cfg.CookieDomain, cfg.TrustedProxyHops,
	)

	// ルーティング（公開 API は /api/v1/...）
	http.HandleFunc("/", handleRoot)
	http.HandleFunc(availabilityPath, availabilityHandler.HandleAvailability)
	http.HandleFunc(publicSchedulesPath, publicScheduleHandler.HandlePublicSchedules)
	http.HandleFunc(reservationsPath, reservationHandler.HandleReservations)
	http.HandleFunc(publicSuppliersPath, publicSupplierHandler.HandlePublicSuppliers)

	if cfg.OutboxFlushEndpointEnabled {
		outboxHandler := handler.NewOutboxHandler(mailDispatcher)
		http.HandleFunc("/internal/outbox/flush", outboxHandler.HandleFlush)
		log.Println("Outbox flush endpoint enabled at POST /internal/outbox/flush")
	}

	// 認証エンドポイント、login / refresh は AT 不要
	http.HandleFunc(adminBase+"/login", authHandler.HandleLogin)
	http.HandleFunc(adminBase+"/refresh", authHandler.HandleRefresh)
	// logout は有効な AT 必須 → RequireAuth でラップ
	http.Handle(adminBase+"/logout", handler.RequireAuth(jwtService)(http.HandlerFunc(authHandler.HandleLogout)))

	scheduleAuth := handler.RequireAuth(jwtService)
	http.Handle(schedulesPath, scheduleAuth(http.HandlerFunc(scheduleHandler.HandleSchedules)))
	http.Handle(schedulesPath+"/", scheduleAuth(http.HandlerFunc(scheduleHandler.HandleSchedules)))

	http.Handle(adminReservationsPath, scheduleAuth(http.HandlerFunc(adminReservationHandler.HandleAdminReservations)))
	http.Handle(adminReservationsPath+"/", scheduleAuth(http.HandlerFunc(adminReservationHandler.HandleAdminReservations)))

	http.Handle(adminSuppliersPath, scheduleAuth(http.HandlerFunc(adminSupplierHandler.HandleAdminSuppliers)))
	http.Handle(adminSuppliersPath+"/", scheduleAuth(http.HandlerFunc(adminSupplierHandler.HandleAdminSuppliers)))

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

	go listenForShutdown(srv)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func listenForShutdown(srv *http.Server) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
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
