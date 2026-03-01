package config

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env         string `yaml:"env" env-default:"local"`
	StorageType string `yaml:"storage_type" env-default:"sqlite"`
	StoragePath string `yaml:"storage_path"`
	Postgres    `yaml:"postgres"`
	HTTPServer  `yaml:"http_server"`
}

type Postgres struct {
	DSN      string `yaml:"dsn"`
	Host     string `yaml:"host" env-default:"localhost"`
	Port     int    `yaml:"port" env-default:"5432"`
	User     string `yaml:"user" env-default:"postgres"`
	Password string `yaml:"password" env-default:"postgres"`
	DBName   string `yaml:"dbname" env-default:"url_shortener"`
	SSLMode  string `yaml:"sslmode" env-default:"disable"`
}

func (p Postgres) ConnString() string {
	if p.DSN != "" {
		return p.DSN
	}

	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host,
		p.Port,
		p.User,
		p.Password,
		p.DBName,
		p.SSLMode,
	)
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:"localhost:8082"`
	Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		// Backward compatibility for old env name.
		configPath = os.Getenv("CONFIG_PAH")
	}
	//configPath как его установить при запуске?
	//  export CONFIG_PATH=./config/local.yaml
	// CONFIG_PATH=./config/postgres.yaml go run cmd/url-shortener/main.go

	if configPath == "" {
		log.Fatal()
	}
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file not found: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("failed to read config: %v", err)

	}
	return &cfg
}
