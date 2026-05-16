package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort            string
	BrokerType            string
	KafkaBootstrapServers string
	KafkaGroupID          string
	BrevoAPIKey           string
	MailFrom              string
	EurekaServerURL       string
	AppBaseURL            string
}

func LoadConfig() *Config {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: .env file not found, using system env variables")
	}

	return &Config{
		ServerPort:            viper.GetString("SERVER_PORT"),
		BrokerType:            viper.GetString("BROKER_TYPE"),
		KafkaBootstrapServers: viper.GetString("KAFKA_BOOTSTRAP_SERVERS"),
		KafkaGroupID:          viper.GetString("KAFKA_CONSUMER_GROUP_ID"),
		BrevoAPIKey:           viper.GetString("BREVO_API_KEY"),
		MailFrom:              viper.GetString("MAIL_FROM"),
		EurekaServerURL:       viper.GetString("EUREKA_SERVER_URL"),
		AppBaseURL:            viper.GetString("APP_BASE_URL"),
	}
}
