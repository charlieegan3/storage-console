package handlers

import "log"

type Options struct {
	DevMode    bool
	EtagScript string
	EtagStyles string

	LoggerError *log.Logger
	LoggerInfo  *log.Logger
}
