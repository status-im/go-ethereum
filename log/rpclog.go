package log

import (
	"sync/atomic"
)

var (
	rpcLogger Logger
	// isRPCLoggingEnabled controls whether RPC logging is enabled
	isRPCLoggingEnabled uint32
)

// NewRPCLogger creates a new RPC logger object
func NewRPCLogger(ctx ...interface{}) Logger {
	rpcLogger = root.New(append([]interface{}{"module", "rpc"}, ctx...)...)
	return rpcLogger
}

// EnableRPCLogging enables or disables RPC logging
func EnableRPCLogging(enable bool) {
	if enable {
		atomic.StoreUint32(&isRPCLoggingEnabled, 1)
	} else {
		atomic.StoreUint32(&isRPCLoggingEnabled, 0)
	}
}

// IsRPCLoggingEnabled returns whether RPC logging is enabled
func IsRPCLoggingEnabled() bool {
	return atomic.LoadUint32(&isRPCLoggingEnabled) == 1
}

// GetRPCLogger returns the RPC logger object
func GetRPCLogger() Logger {
	return rpcLogger
}
