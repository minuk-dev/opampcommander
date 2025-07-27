package client

import (
	"fmt"
	"log/slog"
	"runtime"

	"github.com/go-resty/resty/v2"
)

var _ resty.Logger = (*loggerWrapper)(nil)

type loggerWrapper struct {
	*slog.Logger
}

//nolint:mnd
func (logger *loggerWrapper) Errorf(format string, v ...interface{}) {
	_, f, l, _ := runtime.Caller(2)
	source := slog.String("originSource", fmt.Sprintf("%s:%d", f, l))
	logger.With(source).Error(fmt.Sprintf(format, v...))
}

//nolint:mnd
func (logger *loggerWrapper) Warnf(format string, v ...interface{}) {
	_, f, l, _ := runtime.Caller(2)
	source := slog.String("originSource", fmt.Sprintf("%s:%d", f, l))
	logger.With(source).Warn(fmt.Sprintf(format, v...))
}

//nolint:mnd
func (logger *loggerWrapper) Debugf(format string, v ...interface{}) {
	_, f, l, _ := runtime.Caller(2)
	source := slog.String("originSource", fmt.Sprintf("%s:%d", f, l))
	logger.With(source).Debug(fmt.Sprintf(format, v...))
}
