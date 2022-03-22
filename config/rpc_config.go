package config

import "time"

const (
	// ServerMinInterval server min interval
	ServerMinInterval = time.Duration(1) * time.Minute
	// ConnectionTimeout connection timeout time
	ConnectionTimeout = 5 * time.Second
)
