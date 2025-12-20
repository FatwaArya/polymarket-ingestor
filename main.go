package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // Enable pprof for Roumon
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/FatwaArya/pm-ingest/config"
	"github.com/FatwaArya/pm-ingest/internal"
	"github.com/FatwaArya/pm-ingest/internal/domain"
	internalkafka "github.com/FatwaArya/pm-ingest/internal/kafka"
	"github.com/FatwaArya/pm-ingest/utils"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Printf("Starting application in %s mode on port %s", config.AppConfig.GinMode, config.AppConfig.AppPort)
	log.Printf("Kafka brokers: %s, topic: %s", config.AppConfig.KafkaBrokers, config.AppConfig.KafkaTopic)

	var processedTrades uint64
	verbose := true

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx := context.Background()

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

	// Kafka producer for trades
	kafkaBrokers := strings.TrimSpace(config.AppConfig.KafkaBrokers)
	producer, err := internalkafka.NewProducer(kafkaBrokers, config.AppConfig.KafkaTopic)
	if err != nil {
		log.Fatalf("failed to create kafka producer: %v", err)
	}
	defer producer.Close()

	// Discovery service consumer for high-value traders
	discoveryService, err := domain.NewDiscoveryService(
		kafkaBrokers,
		config.AppConfig.KafkaTopic,
		"discovery-service-group", // Consumer group ID
	)
	if err != nil {
		log.Fatalf("failed to create discovery service: %v", err)
	}
	defer discoveryService.Close()

	// Run discovery service in a goroutine
	go func() {
		log.Println("Starting discovery service consumer...")
		if err := discoveryService.Run(ctx); err != nil {
			log.Printf("Discovery service error: %v", err)
		}
	}()

	// // Confidence service for calculating user confidence based on new bets and closed positions
	// confidenceService, err := domain.NewConfidenceService(
	// 	kafkaBrokers,
	// 	config.AppConfig.KafkaTopic,
	// 	"confidence-service-group", // Consumer group ID
	// )
	// if err != nil {
	// 	log.Fatalf("failed to create confidence service: %v", err)
	// }
	// defer confidenceService.Close()

	// // Run confidence service in a goroutine
	// go func() {
	// 	log.Println("Starting confidence service consumer...")
	// 	if err := confidenceService.Run(ctx); err != nil {
	// 		log.Printf("Confidence service error: %v", err)
	// 	}
	// }()

	// Create WebSocket client
	client := internal.NewWebSocketClient(
		subscriptions,
		func(message []byte) {
			// print raw and parsed

			trade, err := utils.ParseActivityTrade(message)
			if err != nil {
				// Skip non-trade messages silently
				if errors.Is(err, utils.ErrSkipMessage) {
					return
				}
				log.Printf("Error parsing activity trade: %v", err)
				return
			}

			if err := producer.ProduceTrade(ctx, trade); err != nil {
				log.Printf("Error producing trade to Kafka for id=%s: %v", trade.TransactionHash, err)
				return
			}
			if verbose {
				count := atomic.AddUint64(&processedTrades, 1)
				if count%100 == 0 {
					log.Printf("Processed trades: %d", count)
				}
			}
		},
		verbose,
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
