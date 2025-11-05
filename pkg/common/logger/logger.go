package logger

import (
	"github.com/sirupsen/logrus"
	"os"
)

var Log *logrus.Logger

func Init() {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})

	level := os.Getenv("LOG_LEVEL")
	if level == "" {
		level = "info"
	}

	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	Log.SetLevel(logLevel)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return Log.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}

