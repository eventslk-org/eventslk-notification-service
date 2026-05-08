package config

import (
	"log"
	"github.com/spf13/viper"
)

type Config struct {
	ServerPort           string
	KafkaBootstrapServers string
	KafkaGroupID         string
	MailHost             string
	MailPort             int
	MailUsername         string
	MailPassword         string
	MailFrom             string
	EurekaServerURL      string
	AppBaseURL           string
}

func LoadConfig() *Config {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: .env file not found, using system env variables")
	}

	return &Config{
		ServerPort:            viper.GetString("SERVER_PORT"),
		KafkaBootstrapServers: viper.GetString("KAFKA_BOOTSTRAP_SERVERS"),
		KafkaGroupID:          viper.GetString("KAFKA_CONSUMER_GROUP_ID"),
		MailHost:              viper.GetString("MAIL_HOST"),
		MailPort:              viper.GetInt("MAIL_PORT"),
		MailUsername:          viper.GetString("MAIL_USERNAME"),
		MailPassword:          viper.GetString("MAIL_PASSWORD"),
		MailFrom:              viper.GetString("MAIL_FROM"),
		EurekaServerURL:       viper.GetString("EUREKA_SERVER_URL"),
		AppBaseURL:            viper.GetString("APP_BASE_URL"),
	}
}