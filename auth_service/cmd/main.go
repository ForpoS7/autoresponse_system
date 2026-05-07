package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"auth_service/internal/config"
	"auth_service/internal/handler"
	"auth_service/internal/jwt"
	"auth_service/internal/repository"
	"auth_service/internal/service"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func main() {
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name,
		cfg.Database.User, cfg.Database.Password, cfg.Database.SSLMode,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db open error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("db ping error: %v", err)
	}
	log.Println("connected to PostgreSQL")

	jwtManager := jwt.NewManager(cfg.JWT.Secret, cfg.JWT.Expiration)
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, jwtManager)
	authHandler := handler.NewAuthHandler(authSvc)

	r := mux.NewRouter()
	r.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	r.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)
	r.HandleFunc("/validate", authHandler.Validate).Methods(http.MethodPost)
	r.HandleFunc("/health", authHandler.Health).Methods(http.MethodGet)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("auth_service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
