package main

import (
	"log"
	"os"

	app "external-payment-service/internal/app"
	"external-payment-service/internal/config"
)

const (
	example     = "example"
	development = "development"
	production  = "production"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = example
	}

	cfg, err := config.LoadConfig(env)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	server, err := app.NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
