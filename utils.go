package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/bagaking/goulp/wlog"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

func mustInitLogger() {
	// 配置一个lumberjack.Logger
	logRoller := &lumberjack.Logger{
		Filename:   "./botheater.log", // 日志文件的位置
		MaxSize:    10,                // 日志文件的最大大小（MB）
		MaxBackups: 31,                // 保存的旧日志文件最大个数
		MaxAge:     31,                // 保存的旧日志文件的最大天数
		// Compress:   true,              // 是否压缩归档的日志文件
	}
	defer func() {
		if err := logRoller.Close(); err != nil {
			fmt.Println("Failed to close log", err)
		}
	}()

	multiLogger := io.MultiWriter(os.Stdout, logRoller)
	logrus.SetOutput(multiLogger)
	logrus.SetLevel(logrus.DebugLevel) // 设置日志记录级别

	// 设置 TextFormatter 以保留换行和颜色
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true, // 强制颜色输出
		FullTimestamp: true, // 显示完整时间戳
	})

	// Gin 设置
	wlog.SetEntryGetter(
		func(ctx context.Context) *logrus.Entry {
			return logrus.WithContext(ctx)
		},
	)
}
