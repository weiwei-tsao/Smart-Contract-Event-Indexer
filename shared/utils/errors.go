package utils

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AppError represents an application error with additional context
type AppError struct {
	Code    ErrorCode
	Message string
	Err     error
	Context map[string]interface{}
}

// ErrorCode represents application error codes
type ErrorCode string

const (
	// Generic errors
	ErrCodeInternal      ErrorCode = "INTERNAL_ERROR"
	ErrCodeInvalidInput  ErrorCode = "INVALID_INPUT"
	ErrCodeNotFound      ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists ErrorCode = "ALREADY_EXISTS"
	ErrCodeUnauthorized  ErrorCode = "UNAUTHORIZED"
	ErrCodeForbidden     ErrorCode = "FORBIDDEN"

	// Database errors
	ErrCodeDatabase           ErrorCode = "DATABASE_ERROR"
	ErrCodeDatabaseConnection ErrorCode = "DATABASE_CONNECTION_ERROR"
	ErrCodeDatabaseQuery      ErrorCode = "DATABASE_QUERY_ERROR"

	// RPC errors
	ErrCodeRPC           ErrorCode = "RPC_ERROR"
	ErrCodeRPCConnection ErrorCode = "RPC_CONNECTION_ERROR"
	ErrCodeRPCTimeout    ErrorCode = "RPC_TIMEOUT"

	// Contract errors
	ErrCodeContractNotFound      ErrorCode = "CONTRACT_NOT_FOUND"
	ErrCodeContractAlreadyExists ErrorCode = "CONTRACT_ALREADY_EXISTS"
	ErrCodeInvalidContractABI    ErrorCode = "INVALID_CONTRACT_ABI"

	// Event errors
	ErrCodeEventNotFound    ErrorCode = "EVENT_NOT_FOUND"
	ErrCodeEventParseFailed ErrorCode = "EVENT_PARSE_FAILED"

	// Indexer errors
	ErrCodeIndexerSync ErrorCode = "INDEXER_SYNC_ERROR"
	ErrCodeReorgDetected ErrorCode = "REORG_DETECTED"
)

// Error implements the error interface
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error
func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(code ErrorCode, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	e.Context[key] = value
	return e
}

// WrapError wraps an existing error with application error
func WrapError(code ErrorCode, message string, err error) error {
	if err == nil {
		return nil
	}
	return NewAppError(code, message, err)
}

// ToGRPCError converts an application error to a gRPC error
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	appErr, ok := err.(*AppError)
	if !ok {
		return status.Error(codes.Internal, err.Error())
	}

	code := errorCodeToGRPCCode(appErr.Code)
	return status.Error(code, appErr.Message)
}

// FromGRPCError converts a gRPC error to an application error
func FromGRPCError(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	code := grpcCodeToErrorCode(st.Code())
	return NewAppError(code, st.Message(), err)
}

// errorCodeToGRPCCode maps application error codes to gRPC codes
func errorCodeToGRPCCode(code ErrorCode) codes.Code {
	switch code {
	case ErrCodeInvalidInput:
		return codes.InvalidArgument
	case ErrCodeNotFound, ErrCodeContractNotFound, ErrCodeEventNotFound:
		return codes.NotFound
	case ErrCodeAlreadyExists, ErrCodeContractAlreadyExists:
		return codes.AlreadyExists
	case ErrCodeUnauthorized:
		return codes.Unauthenticated
	case ErrCodeForbidden:
		return codes.PermissionDenied
	case ErrCodeRPCTimeout:
		return codes.DeadlineExceeded
	case ErrCodeDatabaseConnection, ErrCodeRPCConnection:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

// grpcCodeToErrorCode maps gRPC codes to application error codes
func grpcCodeToErrorCode(code codes.Code) ErrorCode {
	switch code {
	case codes.InvalidArgument:
		return ErrCodeInvalidInput
	case codes.NotFound:
		return ErrCodeNotFound
	case codes.AlreadyExists:
		return ErrCodeAlreadyExists
	case codes.Unauthenticated:
		return ErrCodeUnauthorized
	case codes.PermissionDenied:
		return ErrCodeForbidden
	case codes.DeadlineExceeded:
		return ErrCodeRPCTimeout
	case codes.Unavailable:
		return ErrCodeDatabaseConnection
	default:
		return ErrCodeInternal
	}
}

// IsNotFoundError checks if the error is a not found error
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	appErr, ok := err.(*AppError)
	if !ok {
		return false
	}
	return appErr.Code == ErrCodeNotFound ||
		appErr.Code == ErrCodeContractNotFound ||
		appErr.Code == ErrCodeEventNotFound
}

// IsAlreadyExistsError checks if the error is an already exists error
func IsAlreadyExistsError(err error) bool {
	if err == nil {
		return false
	}
	appErr, ok := err.(*AppError)
	if !ok {
		return false
	}
	return appErr.Code == ErrCodeAlreadyExists ||
		appErr.Code == ErrCodeContractAlreadyExists
}

