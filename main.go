package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"eventslk-notification-service/config"
	"eventslk-notification-service/consumer"
	"eventslk-notification-service/email"
	"eventslk-notification-service/eureka"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("[main] config loaded: port=%s kafka=%s", cfg.ServerPort, cfg.KafkaBootstrapServers)

	emailSender := email.NewEmailSender(
		cfg.MailHost,
		cfg.MailPort,
		cfg.MailUsername,
		cfg.MailPassword,
		cfg.MailFrom,
		"templates",
	)

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
	router.GET("/actuator/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
	})

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
