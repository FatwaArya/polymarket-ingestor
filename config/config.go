package config

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

type Config struct {
	AppPort              string
	GinMode              string
	QuestDBHost          string
	QuestDBILPPort       string
	PolymarketAPIKey     string
	ChainID              string
	PolymarketSecret     string
	PolymarketPassphrase string
}

// global
var AppConfig Config

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Reading configuration from environment variables.")
	}

	AppConfig = Config{
		AppPort:              getEnv("APP_PORT", "8080"),    // Default to 8080
		GinMode:              getEnv("GIN_MODE", "release"), // Default to release
		QuestDBHost:          getEnv("QUESTDB_HOST", "localhost"),
		QuestDBILPPort:       getEnv("QUESTDB_ILP_PORT", "9009"),
		PolymarketAPIKey:     getEnv("POLYMARKET_APIKEY", ""),
		ChainID:              getEnv("CHAIN_ID", "137"),
		PolymarketSecret:     getEnv("POLYMARKET_SECRET", ""),
		PolymarketPassphrase: getEnv("POLYMARKET_PASSPHRASE", ""),
	}

	if AppConfig.PolymarketAPIKey == "" {
		log.Fatal("POLYMARKET_APIKEY is not set")
	}
	if AppConfig.PolymarketSecret == "" {
		log.Fatal("POLYMARKET_SECRET is not set")
	}
	if AppConfig.PolymarketPassphrase == "" {
		log.Fatal("POLYMARKET_PASSPHRASE is not set")
	}

	gin.SetMode(AppConfig.GinMode)
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
