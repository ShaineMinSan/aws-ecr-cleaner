// aws-ecr-cleaner/internal/config/config.go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 保存所有配置信息
type Config struct {
	LogDir            string
	LogFilePath       string
	Debug             bool
	DryRun            bool
	ListOnly          bool
	ProtectLatest     int
	ProtectInUseByK8s bool
	TargetRepoRegex   string
	ExcludeRepoRegex  string
	HoldTagRegex      string
	AWSRegion         string
	Env               string
	ImageListFile     string
	AutoConfirm       bool // 如果为 true，则跳过交互确认直接删除
	InteractiveMode   bool // 如果为 true，则保留终端输出，用于交互提示
}

func LoadConfig() *Config {
	logDir := os.Getenv("LOGDIR")
	if logDir == "" {
		panic("LOGDIR must be set in .env")
	}

	debug := os.Getenv("DEBUG") == "true"
	dryRun := os.Getenv("DRYRUN") == "true"
	listOnly := os.Getenv("LIST_ONLY") == "true"

	protectLatest := 3
	if v := os.Getenv("PROTECT_LATEST"); v != "" {
		if num, err := strconv.Atoi(v); err == nil {
			protectLatest = num
		}
	}

	protectInUseByK8s := os.Getenv("PROTECT_INUSE_BY_K8S") == "true"
	targetRepoRegex := os.Getenv("TARGET_REPO_REGEX")
	holdTagRegex := os.Getenv("HOLD_TAG_REGEX")
	excludeRepoRegex := os.Getenv("EXCLUDE_REPO_REGEX")

	if targetRepoRegex == "" || holdTagRegex == "" {
		panic("TARGET_REPO_REGEX and HOLD_TAG_REGEX must be set in .env")
	}

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		panic("AWS_REGION must be set in environment")
	}

	envVal := strings.ToLower(os.Getenv("ENV"))
	if envVal == "" {
		panic("ENV must be set (pre, prd, mgmt)")
	}
	var imageListFile string
	switch envVal {
	case "pre":
		imageListFile = "IMG_LIST/PRE_IMG_LIST.txt"
	case "prd":
		imageListFile = "IMG_LIST/PRD_IMG_LIST.txt"
	case "mgmt":
		imageListFile = "IMG_LIST/MGMT_IMG_LIST.txt"
	default:
		panic(fmt.Sprintf("Invalid ENV value '%s'. Must be one of: pre, prd, mgmt", envVal))
	}

	timestamp := time.Now().Format("20060102_150405")
	logFilename := fmt.Sprintf("ecr_cleaner_app_%s.log", timestamp)
	logFilePath := filepath.Join(logDir, logFilename)

	// 新增两个配置项：
	autoConfirm := os.Getenv("AUTO_CONFIRM") == "true"           // 若为 true，则自动确认删除
	interactiveMode := os.Getenv("INTERACTIVE_MODE") == "true" // 若为 true，则在终端保留输出，便于交互

	return &Config{
		LogDir:            logDir,
		LogFilePath:       logFilePath,
		Debug:             debug,
		DryRun:            dryRun,
		ListOnly:          listOnly,
		ProtectLatest:     protectLatest,
		ProtectInUseByK8s: protectInUseByK8s,
		TargetRepoRegex:   targetRepoRegex,
		ExcludeRepoRegex:  excludeRepoRegex,
		HoldTagRegex:      holdTagRegex,
		AWSRegion:         awsRegion,
		Env:               envVal,
		ImageListFile:     imageListFile,
		AutoConfirm:       autoConfirm,
		InteractiveMode:   interactiveMode,
	}
}
