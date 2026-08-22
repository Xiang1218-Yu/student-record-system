// Package logger configures the process-wide structured logger (Zap).
// Only this package should construct zap.Logger instances; every other
// package receives a *zap.Logger and writes to it.
package logger

import (
	"go.uber.org/zap"
)

// New constructs a production logger when production mode is requested and a
// development logger otherwise.
func New(production bool) (*zap.Logger, error) {
	if production {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
