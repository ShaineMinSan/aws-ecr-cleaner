package logger

import (
	"log"
	"os"
)

// InitLogger 初始化日志系统，根据 interactiveMode 决定是否重定向标准输出
func InitLogger(logFilePath string, interactiveMode bool) {
	f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file %s: %v", logFilePath, err)
	}
	// 如果不是交互模式，则重定向标准输出和错误输出到日志文件
	if !interactiveMode {
		os.Stdout = f
		os.Stderr = f
	}
	log.SetOutput(f)
	log.Printf("Logging initialized. Log file: %s\n", logFilePath)
}
