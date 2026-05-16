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

	"eventslk-notification-service/broker"
	"eventslk-notification-service/config"
	"eventslk-notification-service/consumer"
	"eventslk-notification-service/email"
	"eventslk-notification-service/eureka"
	"eventslk-notification-service/handler"

	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
)

//go:embed templates
var templatesDir embed.FS

func main() {
	cfg := config.LoadConfig()
	if cfg.BrevoAPIKey == "" {
		log.Println("[DEBUG] CRITICAL: Brevo API Key is EMPTY!")
	} else {
		log.Printf("[DEBUG] Brevo API Key loaded. Length: %d, Prefix: %s...", len(cfg.BrevoAPIKey), cfg.BrevoAPIKey[:10])
	}
	log.Printf("[main] config loaded: port=%s broker=%s kafka=%s", cfg.ServerPort, cfg.BrokerType, cfg.KafkaBootstrapServers)

	templates, err := fs.Sub(templatesDir, "templates")
	if err != nil {
		log.Fatalf("[main] failed to create template sub-FS: %v", err)
	}

	emailSender := email.NewEmailSender(cfg.BrevoAPIKey, cfg.MailFrom, templates)

	ctx, cancel := context.WithCancel(context.Background())

	selection := broker.New(cfg.BrokerType, cfg.KafkaBootstrapServers, cfg.KafkaGroupID,
		func() sarama.ConsumerGroupHandler {
			return consumer.NewNotificationHandler(emailSender)
		})
	log.Printf("[main] broker selected: requested=%s active=%s", selection.Requested, selection.Active)

	if err := selection.Broker.Connect(ctx); err != nil {
		log.Printf("[main] broker connect returned error (continuing): %v", err)
	}
	if err := selection.Broker.Consume(ctx, consumer.Topics); err != nil {
		log.Printf("[main] broker consume returned error (continuing): %v", err)
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
		log.Printf("[main] eureka setup error (registration loop not started): %v", err)
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
	if err := selection.Broker.Close(); err != nil {
		log.Printf("[main] error closing broker: %v", err)
	}
	if eurekaStopCh != nil {
		close(eurekaStopCh)
	}

	log.Println("[main] shutdown complete")
}
