package filter

import (
	"github.com/sirupsen/logrus"
)

type defaultAccessLogWriter struct{}

func (w *defaultAccessLogWriter) Write(accessLog *AccessLog) {
	logEntry := logrus.WithFields(logrus.Fields{
		"time":       accessLog.Timestamp,
		"remote_ip":  accessLog.RemoteIP,
		"host":       accessLog.Host,
		"method":     accessLog.Method,
		"uri":        accessLog.Uri,
		"user_agent": accessLog.UserAgent,
		"status":     accessLog.Status,
		"error":      accessLog.Error,
		"latency":    accessLog.Latency,
		"bytes_in":   accessLog.BytesSent,
	})

	if accessLog.Error == "" {
		logEntry.Info("accesslog")
	} else {
		logEntry.Error("accesslog")
	}

}
