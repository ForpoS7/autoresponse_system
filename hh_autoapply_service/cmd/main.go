package main

import (
	"context"
	"database/sql"
	"fmt"
	"hh_autoapply_service/internal/config"
	"hh_autoapply_service/internal/handler"
	"hh_autoapply_service/internal/repository"
	"hh_autoapply_service/internal/service"
	"hh_autoapply_service/pkg/ai"
	"hh_autoapply_service/pkg/httpclient"
	"hh_autoapply_service/pkg/kafka"
	"hh_autoapply_service/pkg/playwright"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting hh_autoapply_service on port %d", cfg.Server.Port)

	db, err := repository.NewDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := initSchema(db); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	kafkaProducer := kafka.NewProducer(cfg.Kafka.Brokers)
	defer kafkaProducer.Close()

	kafkaConsumer := kafka.NewConsumer(cfg.Kafka.Brokers, cfg.Kafka.Topic.Vacancies, "autoapply-service-group")
	defer kafkaConsumer.Close()

	hhClient := httpclient.NewHHAggregateClient("http://localhost:8080", 30*time.Second)

	browserManager, err := playwright.NewBrowserManager(
		cfg.Playwright.Headless,
		cfg.Playwright.SlowMo,
	)
	if err != nil {
		log.Fatalf("Failed to initialize Playwright: %v", err)
	}
	defer browserManager.Close()

	hhTokenRepo := repository.NewHhTokenRepository(db)
	vacancyRepo := repository.NewVacancyRepository(db)
	autoApplyRepo := repository.NewAutoApplyRepository(db)

	vacancyPublisher := service.NewVacancyPublisher(kafkaProducer, cfg.Kafka.Topic.Vacancies)
	playwrightService := service.NewPlaywrightService(browserManager, hhTokenRepo, cfg.Playwright)
	parserService := service.NewParserService(playwrightService, vacancyPublisher, cfg.Playwright.AreaCode)
	tokenService := service.NewTokenService(hhTokenRepo, playwrightService)
	coverLetterService := ai.NewMockCoverLetterService()
	autoApplyService := service.NewAutoApplyService(
		parserService,
		playwrightService,
		coverLetterService,
		autoApplyRepo,
		vacancyRepo,
		hhTokenRepo,
		kafkaConsumer,
		hhClient,
		cfg.JWT.JavaServiceToken,
	)

	tokenHandler := handler.NewTokenHandler(tokenService)
	autoApplyHandler := handler.NewAutoApplyHandler(autoApplyService)

	r := mux.NewRouter()

	r.HandleFunc("/api/hh-token", tokenHandler.GetHHToken).Methods("GET")
	r.HandleFunc("/api/hh-token", tokenHandler.ExtractHHToken).Methods("POST")

	r.HandleFunc("/api/autoapply", autoApplyHandler.CreateAutoApply).Methods("POST")
	r.HandleFunc("/api/autoapply/{id}", autoApplyHandler.GetAutoApplyStatus).Methods("GET")

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server is running on port %d", cfg.Server.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

func initSchema(db *sql.DB) error {
	schema, err := os.ReadFile("internal/repository/schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	_, err = db.ExecContext(context.Background(), string(schema))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	log.Println("Database schema initialized")
	return nil
}
