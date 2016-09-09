package main

import "github.com/Sirupsen/logrus"

type Logger interface {
	Debugf(string, ...interface{})
	Infof(string, ...interface{})
	Warnf(string, ...interface{})
}

type LogrusLogger struct{}

func (ll LogrusLogger) Debugf(str string, args ...interface{}) {
	logrus.Debugf(str, args...)
}

func (ll LogrusLogger) Infof(str string, args ...interface{}) {
	logrus.Infof(str, args...)
}

func (ll LogrusLogger) Warnf(str string, args ...interface{}) {
	logrus.Warnf(str, args...)
}

type System struct {
	Logger
	ConfigGetter
}

func NewSystem() (*System, error) {
	conf, err := NewDefaultConfigClient()
	if err != nil {
		return nil, err
	}

	return &System{
		Logger:       &LogrusLogger{},
		ConfigGetter: conf,
	}, nil
}
