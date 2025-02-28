package main

import (
	"log"

	"github.com/joho/godotenv"
	"aws-ecr-cleaner/internal/cleaner"
	"aws-ecr-cleaner/internal/config"
	"aws-ecr-cleaner/internal/logger"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	cfg := config.LoadConfig()

	// 传入 cfg.InteractiveMode 控制是否保留终端输出
	logger.InitLogger(cfg.LogFilePath, cfg.InteractiveMode)

	cleaner.Run(cfg)
}
