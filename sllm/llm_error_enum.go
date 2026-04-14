package sllm

import "net/http"

// EnumLLMErrorType is a string type for representing LLM provider error constants.
type EnumLLMErrorType string

// Enumeration of common LLM API errors as string type constants.
const (
	ErrLLMContextLengthExceeded  EnumLLMErrorType = "Context Length Exceeded"
	ErrLLMContentPolicyViolation EnumLLMErrorType = "Content Policy Violation"
	ErrLLMInvalidAPIKey          EnumLLMErrorType = "Invalid API Key"
	ErrLLMModelNotFound          EnumLLMErrorType = "Model Not Found"
	ErrLLMQuotaExceeded          EnumLLMErrorType = "Quota Exceeded"
	ErrLLMRateLimitExceeded      EnumLLMErrorType = "Rate Limit Exceeded"
	ErrLLMServiceUnavailable     EnumLLMErrorType = "Service Unavailable"
	ErrLLMTimeout                EnumLLMErrorType = "Timeout"
)

// String returns the string representation of the EnumLLMErrorType.
func (e EnumLLMErrorType) String() string {
	return string(e)
}

// validLLMErrs is a set of all valid EnumLLMErrorType values.
var validLLMErrs = map[EnumLLMErrorType]struct{}{
	ErrLLMContextLengthExceeded:  {},
	ErrLLMContentPolicyViolation: {},
	ErrLLMInvalidAPIKey:          {},
	ErrLLMModelNotFound:          {},
	ErrLLMQuotaExceeded:          {},
	ErrLLMRateLimitExceeded:      {},
	ErrLLMServiceUnavailable:     {},
	ErrLLMTimeout:                {},
}

// Valid checks whether the EnumLLMErrorType is one of the defined constants.
func (e EnumLLMErrorType) Valid() bool {
	_, ok := validLLMErrs[e]
	return ok
}

var llmErrToHTTPStatusMap = map[EnumLLMErrorType]int{
	ErrLLMContextLengthExceeded:  http.StatusBadRequest,
	ErrLLMContentPolicyViolation: http.StatusUnprocessableEntity,
	ErrLLMInvalidAPIKey:          http.StatusUnauthorized,
	ErrLLMModelNotFound:          http.StatusNotFound,
	ErrLLMQuotaExceeded:          http.StatusPaymentRequired,
	ErrLLMRateLimitExceeded:      http.StatusTooManyRequests,
	ErrLLMServiceUnavailable:     http.StatusServiceUnavailable,
	ErrLLMTimeout:                http.StatusGatewayTimeout,
}

// HTTPStatus translates the EnumLLMErrorType to an HTTP status code.
func (e EnumLLMErrorType) HTTPStatus() int {
	if status, ok := llmErrToHTTPStatusMap[e]; ok {
		return status
	}
	return http.StatusInternalServerError
}
