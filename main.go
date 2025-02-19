package main

import (
	"log"

	"github.com/caarlos0/env/v11"
)

type config struct {
	SubscriptionID string `env:"AZURE_SUBSCRIPTION_ID,required"`
	ResourceGroup  string `env:"AZURE_RESOURCE_GROUP,required"`
	SessionPool    string `env:"AZURE_SESSION_POOL,required"`
	Region         string `env:"AZURE_REGION,required"`
	baseURL        string
}

func main() {
	var cfg config
	err := env.Parse(&cfg)
	if err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}
	u, err := baseURL(cfg)
	if err != nil {
		log.Fatalf("failed to get base URL: %v", err)
	}
	cfg.baseURL = u

	s := NewServer(cfg)
	s.Start()
}
