package errutil

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// IsNotFound returns true if the error was gRPC not found error
func IsNotFound(err error) bool {
	return HasCode(err, codes.NotFound)
}

// HasCode returns true if the error has the code
func HasCode(err error, code codes.Code) bool {
	serr, ok := status.FromError(err)
	if ok {
		return serr.Code() == code
	}
	return false
}
