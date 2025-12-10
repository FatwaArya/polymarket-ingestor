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
	"github.com/FatwaArya/pm-ingest/utils"
	"github.com/gin-gonic/gin"
)

func main() {
	log.Printf("Starting application in %s mode on port %s", config.AppConfig.GinMode, config.AppConfig.AppPort)
	log.Printf("QuestDB Connection: %s:%s", config.AppConfig.QuestDBHost, config.AppConfig.QuestDBILPPort)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// assetsIDs := []string{
	// 	// "81104637750588840860328515305303028259865221573278091453716127842023614249200",
	// 	// "60487116984468020978247225474488676749601001829886755968952521846780452448915",
	// }

	// conditionIDs := []string{
	// Add specific condition IDs here, or leave empty for all markets
	// Example: "0x1234..."
	// "0x4319532e181605cb15b1bd677759a3bc7f7394b2fdf145195b700eeaedfd5221",
	// }

	// Auth credentials from config
	auth := &internal.Auth{
		APIKey:     config.AppConfig.PolymarketAPIKey,
		Secret:     config.AppConfig.PolymarketSecret,
		Passphrase: config.AppConfig.PolymarketPassphrase,
	}

	// Create user WebSocket connection (receives your order updates)
	userConn := internal.NewWebSocketOrderBook(
		internal.UserChannel,
		internal.WsURL,
		make([]string, 0),
		auth,
		func(message []byte) {
			msg, err := utils.ParseUserMessage(message)
			if err != nil {
				log.Printf("Failed to parse message: %v", err)
				return
			}

			switch m := msg.(type) {
			case *utils.TradeMessage:
				log.Printf("[TRADE] %s | %s %s @ %s | Status: %s | Market: %s",
					m.ID, m.Side, m.Size, m.Price, m.Status, m.Market)
			case *utils.OrderMessage:
				log.Printf("[ORDER] %s | %s %s @ %s | Type: %s | Market: %s",
					m.ID, m.Side, m.OriginalSize, m.Price, m.Type, m.Market)
			}
		},
		true, // verbose
	)

	// marketConn := internal.NewWebSocketOrderBook(
	// 	internal.MarketChannel,
	// 	internal.WsURL,
	// 	assetsIDs,
	// 	auth,
	// 	func(message []byte) {
	// 		fmt.Println(string(message))
	// 	},
	// 	true, // verbose
	// )

	// Run WebSocket in a goroutine
	go func() {
		if err := userConn.Run(); err != nil {
			log.Printf("User WebSocket error: %v", err)
		}
	}()

	// go func() {
	// 	if err := marketConn.Run(); err != nil {
	// 		log.Printf("Market WebSocket error: %v", err)
	// 	}
	// }()

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
	// marketConn.Close()
	userConn.Close()
}
