package tracker

import (
	"ndiff"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Service struct {
	logger *zap.SugaredLogger
}

func New() *Service {
	var svc Service

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      true,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	if ndiff.LOG_LEVEL == ndiff.LOG_LEVEL_DEBUG {
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	svc.logger = logger.Sugar()
	return &svc
}

func (l *Service) Debug(eventType string, message string, kv ...ndiff.Tag) {
	l.with(eventType, kv...).Debugf(message)
}
func (l *Service) Info(eventType string, message string, kv ...ndiff.Tag) {
	l.with(eventType, kv...).Infof(message)
}
func (l *Service) Warn(eventType string, message string, kv ...ndiff.Tag) {
	l.with(eventType, kv...).Warnf(message)
}
func (l *Service) Error(eventType string, message string, kv ...ndiff.Tag) {
	l.with(eventType, kv...).Errorf(message)
}
func (l *Service) Fatal(eventType string, message string, kv ...ndiff.Tag) {
	l.with(eventType, kv...).Fatalf(message)
}
func (l *Service) Flush() {
	_ = l.logger.Sync()
}

func (l *Service) with(eventType string, kv ...ndiff.Tag) *zap.SugaredLogger {
	tmp := l.logger.With("event_type", eventType)
	for _, tag := range kv {
		tmp = tmp.With(tag.Key, tag.Value)
	}
	return tmp
}
