package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"eventslk-notification-service/config"
	"eventslk-notification-service/consumer"
	"eventslk-notification-service/email"
	"eventslk-notification-service/eureka"
	"eventslk-notification-service/handler"

	"github.com/gin-gonic/gin"
)

//go:embed templates
var templatesDir embed.FS

func main() {
	cfg := config.LoadConfig()
	// Add this temporary debug block
	if cfg.BrevoAPIKey == "" {
    	log.Println("[DEBUG] CRITICAL: Brevo API Key is EMPTY!")
	} else {
    	// Print the length and first 10 characters safely to verify
    	log.Printf("[DEBUG] Brevo API Key loaded. Length: %d, Prefix: %s...", len(cfg.BrevoAPIKey), cfg.BrevoAPIKey[:10])
	}
	log.Printf("[main] config loaded: port=%s kafka=%s", cfg.ServerPort, cfg.KafkaBootstrapServers)

	// Strip the top-level "templates/" prefix so ParseFS receives bare filenames.
	templates, err := fs.Sub(templatesDir, "templates")
	if err != nil {
		log.Fatalf("[main] failed to create template sub-FS: %v", err)
	}

	emailSender := email.NewEmailSender(cfg.BrevoAPIKey, cfg.MailFrom, templates)

	ctx, cancel := context.WithCancel(context.Background())

	consumerGroup, err := consumer.StartConsumers(ctx, cfg.KafkaBootstrapServers, cfg.KafkaGroupID, emailSender)
	if err != nil {
		log.Fatalf("[main] failed to start Kafka consumers: %v", err)
	}

	instanceID := "eventslk-notification-service:" + cfg.ServerPort
	eurekaStopCh, err := eureka.Register(
		cfg.EurekaServerURL,
		"EVENTSLK-NOTIFICATION-SERVICE",
		instanceID,
		cfg.ServerPort,
		cfg.AppBaseURL,
	)
	if err != nil {
		log.Printf("[main] eureka registration failed (continuing without discovery): %v", err)
	}

	router := gin.Default()
	router.GET("/actuator/health", handler.Health)
	router.GET("/debug/send-test-email", handler.TestEmailHandler(emailSender))

	go func() {
		addr := fmt.Sprintf(":%s", cfg.ServerPort)
		log.Printf("[main] HTTP server listening on %s", addr)
		if err := router.Run(addr); err != nil {
			log.Fatalf("[main] HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("[main] shutdown signal received")

	cancel()
	if err := consumerGroup.Close(); err != nil {
		log.Printf("[main] error closing consumer group: %v", err)
	}
	if eurekaStopCh != nil {
		close(eurekaStopCh)
	}

	log.Println("[main] shutdown complete")
}
