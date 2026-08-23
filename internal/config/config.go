package config

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"log"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	ServerAddr    string `env:"RUN_ADDRESS"`
	AccrualAddr   string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LogLevel      string `env:"LOG"`
	DatabaseURI   string `env:"DATABASE_URI"`
	JWTSecret     string `env:"JWT_SECRET_KEY"`
	SecretKeyByte []byte
	JWTExpire     int64
	JWTExpireTime time.Duration
}

func NewConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on system envs")
	}

	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddr, "a", ":8080", "address to run server (e.g., localhost:8888)")
	flag.StringVar(&cfg.AccrualAddr, "r", "http://localhost:8080", "addres for accural system")
	flag.StringVar(&cfg.LogLevel, "l", "info", "log level")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "DB URI")
	flag.StringVar(&cfg.JWTSecret, "s", "", "secret key for cookie signing (base64)")
	flag.Int64Var(&cfg.JWTExpire, "e", 3600, "expire time for token")
	flag.Parse()

	if err := env.Parse(cfg); err != nil {
		log.Fatal(err)
	}

	cfg.JWTExpireTime = time.Duration(cfg.JWTExpire) * time.Second

	if cfg.JWTSecret != "" {
		key, err := base64.StdEncoding.DecodeString(cfg.JWTSecret)
		if err != nil {
			return nil, err
		}
		cfg.SecretKeyByte = key
	} else {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		cfg.SecretKeyByte = key
	}

	return cfg, nil
}
