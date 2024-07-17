package utils

import (
	"context"
	"fmt"
	"github.com/bagaking/goulp/wlog"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
)

var LogDir = "./logs"

var botEntries = make(map[string]*logrus.Entry)

func MustInitLogger() {
	// 确保日志目录存在
	if err := os.MkdirAll(LogDir, os.ModePerm); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		return
	}

	// 配置一个lumberjack.Logger
	stdLogger := configureLogger(logrus.StandardLogger(), "./botheater.log")

	// Gin 设置
	wlog.SetEntryGetter(
		func(ctx context.Context) *logrus.Entry {
			keyBotLog, ok := ExtractAgentLogKey(ctx)
			if !ok {
				return stdLogger.WithContext(ctx)
			}

			botEntry, ok := botEntries[keyBotLog]
			if !ok || botEntry == nil {
				logFile := fmt.Sprintf("./botheater_%s.log", keyBotLog)
				l := configureLogger(logrus.New(), logFile)
				botEntry = l.WithField("agent", keyBotLog)
				botEntries[keyBotLog] = botEntry
			}

			e := botEntry.WithContext(ctx)

			if id, ok := ExtractAgentID(ctx); ok {
				e = e.WithField("agent_id", id)
			}
			return e
		},
	)
}

// 配置公共的日志设置
func configureLogger(logger *logrus.Logger, outFile string) *logrus.Logger {
	logRoller := &lumberjack.Logger{
		Filename:   filepath.Join(LogDir, outFile),
		MaxSize:    10,
		MaxBackups: 31,
		MaxAge:     31,
	}
	var multiLogger io.Writer
	if logger == logrus.StandardLogger() {
		multiLogger = io.MultiWriter(os.Stdout, logRoller)
	} else {
		multiLogger = io.MultiWriter(logrus.StandardLogger().Out, logRoller)
	}
	logger.SetOutput(multiLogger)
	logger.SetLevel(logrus.DebugLevel) // 设置日志记录级别
	// 设置 TextFormatter 以保留换行和颜色
	logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true, // 强制颜色输出
		FullTimestamp: true, // 显示完整时间戳
	})
	return logger
}

func CountTokens(text string) int {
	// 这里可以引入第三方库或自定义函数来计算 token 数量
	// 例如，使用 BPE 算法或其他分词算法
	// todo: 这里为了简单起见，暂时使用字符数作为 token 数量
	return len([]rune(text))
}

// TakeSentences 从句子列表中取出一段话，并返回剩余的句子列表
// lst 为句子列表, 由原始文本分割得到 (比如根据 \n 分隔)
// maxToken 为每段话的最大 token 数
func TakeSentences(lst []string, maxToken int) (paragraph []rune, left []string) {
	lenSentence := len(lst)
	if lenSentence == 0 {
		return nil, nil
	}
	if lenSentence == 1 {
		return []rune(lst[0]), nil
	}
	stash := lst[0]
	for i := 1; i < lenSentence; i++ {
		sentence := lst[i]
		if CountTokens(stash)+CountTokens(sentence) > maxToken {
			return []rune(stash), lst[i:]
		}
		stash += "\n" + lst[i]
	}

	return []rune(stash), nil
}
