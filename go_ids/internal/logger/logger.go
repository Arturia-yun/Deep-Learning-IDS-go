package logger

import (
	"io"
	"os"
	"path/filepath"

	"go-ids/internal/loader"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Setup 配置全局日志系统
func Setup(cfg loader.LoggingConfig) {
	// 1. 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// 2. 设置日志格式
	if cfg.Format == "json" {
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02 15:04:05",
		})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}

	// 3. 配置输出目标
	var writers []io.Writer

	// 始终输出到控制台 (除非明确禁止，但作为 IDS 实时看很重要)
	// 可以根据 output 字段决定是否只写文件，但通常 both 是最好的
	if cfg.Output == "stdout" || cfg.Output == "both" {
		writers = append(writers, os.Stdout)
	}

	if cfg.Output == "file" || cfg.Output == "both" {
		// 确保日志目录存在
		logDir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			logrus.Errorf("无法创建日志目录: %v", err)
		} else {
			fileLogger := &lumberjack.Logger{
				Filename:   cfg.FilePath,
				MaxSize:    cfg.MaxSize, // MB
				MaxBackups: cfg.MaxBackups,
				MaxAge:     cfg.MaxAge, // days
				Compress:   true,       // 默认压缩旧日志
			}
			writers = append(writers, fileLogger)
		}
	}

	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	logrus.SetOutput(io.MultiWriter(writers...))
}
