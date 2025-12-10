package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Enable pprof for Roumon
	"os"
	"os/signal"
	"syscall"

	"github.com/FatwaArya/pm-ingest/config"
	"github.com/FatwaArya/pm-ingest/internal"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Printf("Starting application in %s mode on port %s", config.AppConfig.GinMode, config.AppConfig.AppPort)
	log.Printf("QuestDB Connection: %s:%s", config.AppConfig.QuestDBHost, config.AppConfig.QuestDBILPPort)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create subscriptions for activity trades (public, no auth needed)
	subscriptions := []internal.Subscription{
		internal.NewActivityTradesSubscription(),
	}

	// Optionally add clob_user subscription if auth is configured
	// if config.AppConfig.PolymarketAPIKey != "" {
	// 	auth := &internal.Auth{
	// 		APIKey:     config.AppConfig.PolymarketAPIKey,
	// 		Secret:     config.AppConfig.PolymarketSecret,
	// 		Passphrase: config.AppConfig.PolymarketPassphrase,
	// 	}
	// 	subscriptions = append(subscriptions, internal.NewClobUserSubscription(auth))
	// }

	// Create WebSocket client
	client := internal.NewWebSocketClient(
		subscriptions,
		func(message []byte) {
			log.Printf("[RAW] %s", string(message))
		},
		true, // verbose
	)

	// Run WebSocket in a goroutine
	go func() {
		if err := client.Run(); err != nil {
			log.Printf("WebSocket error: %v", err)
		}
	}()

	// Setup Gin router
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// Start server in a goroutine
	go func() {
		if err := r.Run(fmt.Sprintf(":%s", config.AppConfig.AppPort)); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Start pprof server for Roumon goroutine monitoring
	go func() {
		log.Println("pprof server running on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()

	log.Printf("Server is running on port %s", config.AppConfig.AppPort)

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")
	client.Close()
}
