package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

func Init() {
	consoleEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder, // 终端彩色
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})

	fileEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	lumberjackLogger := &lumberjack.Logger{
		Filename:   "logs/csu-star.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	consoleOutput := zapcore.Lock(os.Stdout)
	fileOutput := zapcore.AddSync(lumberjackLogger)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleOutput, zap.DebugLevel),
		zapcore.NewCore(fileEncoder, fileOutput, zap.InfoLevel),
	)

	Log = zap.New(core, zap.AddCaller())
}
