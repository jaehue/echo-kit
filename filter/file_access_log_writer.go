package filter

import (
	"os"

	"github.com/fatih/structs"
	"github.com/sirupsen/logrus"
)

type fileLogWriter struct {
	logger *logrus.Logger
}

func (w *fileLogWriter) Write(accessLog *AccessLog) {
	logEntry := w.logger.WithField("level", "")
	for k, v := range structs.New(accessLog).Map() {
		if k == "Id" || v == nil || v == int64(0) || v == "" {
			continue
		}
		if m, ok := v.(map[string]interface{}); ok && len(m) == 0 {
			continue
		}

		logEntry = logEntry.WithField(k, v)
	}
	if accessLog.Error == "" {
		logEntry.Info()
	} else {
		logEntry.Error()
	}
}

func FileLogWriter(filename string) AccessLogWriter {
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.Out = file
	} else {
		logger.Info("Failed to log to file, using default stderr")
	}

	return &fileLogWriter{
		logger: logger,
	}
}
