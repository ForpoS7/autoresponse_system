package main

import (
	"context"
	"database/sql"
	"fmt"
	"hh_autoapply_service/internal/cache"
	"hh_autoapply_service/internal/config"
	"hh_autoapply_service/internal/handler"
	"hh_autoapply_service/internal/middleware"
	"hh_autoapply_service/internal/repository"
	"hh_autoapply_service/internal/service"
	"hh_autoapply_service/pkg/kafka"
	"hh_autoapply_service/pkg/playwright"
	redisclient "hh_autoapply_service/pkg/redis"
	"hh_autoapply_service/pkg/storage"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}

	log.Printf("Starting hh_autoapply_service on port %d", cfg.Server.Port)

	db, err := repository.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	defer db.Close()

	if err := initSchema(db); err != nil {
		log.Fatalf("schema init error: %v", err)
	}

	// ─── Kafka producers ──────────────────────────────────────────────────────
	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer kafkaProducer.Close()

	parsePublisher := kafka.NewParsePublisher(cfg.Kafka.Brokers)
	defer parsePublisher.Close()

	letterPublisher := kafka.NewLetterPublisher(cfg.Kafka.Brokers)
	defer letterPublisher.Close()

	// ─── Redis ───────────────────────────────────────────────────────────────
	// Используется как primary backend для TokenCache и кеша статусов автоотклика.
	// При недоступности Redis сервис продолжает работу (in-memory fallback).
	var rdb *redisclient.Client
	{
		client := redisclient.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err := client.Ping(context.Background()); err != nil {
			log.Printf("WARNING: Redis unavailable (%v) — using in-memory cache only", err)
		} else {
			rdb = client
			log.Printf("Redis connected: %s", cfg.Redis.Addr)
		}
	}

	// ─── Token cache + consumer ───────────────────────────────────────────────
	// Кеш заполняется при старте из Kafka (token.updated).
	// Redis — primary (переживает рестарт), in-memory — fallback.
	tokenCache := cache.NewTokenCache(rdb)
	tokenConsumer := kafka.NewTokenConsumer(cfg.Kafka.Brokers, tokenCache)
	defer tokenConsumer.Close()

	// ─── Correlators (фоновые consumers) ─────────────────────────────────────
	vacancyCorrelator := kafka.NewVacancyCorrelator(cfg.Kafka.Brokers)
	defer vacancyCorrelator.Close()

	letterCorrelator := kafka.NewLetterCorrelator(cfg.Kafka.Brokers)
	defer letterCorrelator.Close()

	// ─── Playwright ───────────────────────────────────────────────────────────
	browserManager, err := playwright.NewBrowserManager(
		cfg.Playwright.Headless,
		cfg.Playwright.SlowMo,
	)
	if err != nil {
		log.Fatalf("playwright init error: %v", err)
	}
	defer browserManager.Close()

	// ─── Repositories ─────────────────────────────────────────────────────────
	hhTokenRepo := repository.NewHhTokenRepository(db)
	vacancyRepo := repository.NewVacancyRepository(db)
	autoApplyRepo := repository.NewAutoApplyRepository(db)

	// ─── Services ─────────────────────────────────────────────────────────────
	vacancyPublisher := service.NewVacancyPublisher(kafkaProducer, cfg.Kafka.Topic.Vacancies)
	playwrightService := service.NewPlaywrightService(browserManager, hhTokenRepo, cfg.Playwright)
	parserService := service.NewParserService(playwrightService, vacancyPublisher, cfg.Playwright.AreaCode)
	tokenService := service.NewTokenService(hhTokenRepo, playwrightService)

	autoApplyService := service.NewAutoApplyService(
		playwrightService,
		autoApplyRepo,
		vacancyRepo,
		hhTokenRepo,
		tokenCache,
		parsePublisher,
		vacancyCorrelator,
		letterPublisher,
		letterCorrelator,
	)

	// ─── Запускаем фоновые consumers ─────────────────────────────────────────
	ctx := context.Background()
	tokenConsumer.Run(ctx)
	vacancyCorrelator.Run(ctx)
	letterCorrelator.Run(ctx)
	log.Println("Background Kafka consumers started")

	// ─── Cold Storage (MinIO) ─────────────────────────────────────────────────
	minioClient, err := storage.NewMinIOClient(
		cfg.Storage.Endpoint,
		cfg.Storage.AccessKey,
		cfg.Storage.SecretKey,
		cfg.Storage.Bucket,
	)
	if err != nil {
		log.Printf("WARNING: MinIO unavailable (%v) — cold storage disabled", err)
		minioClient = nil
	}

	// ─── Archiver ─────────────────────────────────────────────────────────────
	var archiverHandler *handler.ArchiverHandler
	if minioClient != nil {
		archiverSvc := service.NewArchiverService(
			autoApplyRepo,
			vacancyRepo,
			minioClient,
			cfg.Storage.LogsRetainDays,
			cfg.Storage.VacsRetainDays,
		)
		archiverSvc.StartScheduler(ctx, cfg.Storage.ArchiveInterval)
		archiverHandler = handler.NewArchiverHandler(archiverSvc)
		log.Printf("Archiver started: logs>%dd → cold, vacancies>%dd → cold",
			cfg.Storage.LogsRetainDays, cfg.Storage.VacsRetainDays)
	}

	// ─── HTTP роутер ──────────────────────────────────────────────────────────
	r := mux.NewRouter()
	r.Use(middleware.Metrics) // Prometheus HTTP metrics на каждый запрос

	tokenHandler := handler.NewTokenHandler(tokenService)
	r.HandleFunc("/api/hh-token", tokenHandler.GetHHToken).Methods("GET")
	r.HandleFunc("/api/hh-token", tokenHandler.ExtractHHToken).Methods("POST")

	autoApplyHandler := handler.NewAutoApplyHandler(autoApplyService, rdb)
	r.HandleFunc("/api/autoapply", autoApplyHandler.CreateAutoApply).Methods("POST")
	r.HandleFunc("/api/autoapply/{id}", autoApplyHandler.GetAutoApplyStatus).Methods("GET")

	// parserService и parserHandler оставляем для прямого REST-вызова
	_ = parserService

	if archiverHandler != nil {
		r.HandleFunc("/api/archive/run", archiverHandler.RunArchive).Methods("POST")
		r.HandleFunc("/api/archive/list", archiverHandler.ListArchive).Methods("GET")
	}

	// /metrics — Prometheus scrape endpoint (не через nginx, только внутри сети)
	r.Handle("/metrics", promhttp.Handler())

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}).Methods("GET")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server listening on :%d", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func initSchema(db *sql.DB) error {
	schema, err := os.ReadFile("internal/repository/schema.sql")
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	_, err = db.ExecContext(context.Background(), string(schema))
	return err
}
