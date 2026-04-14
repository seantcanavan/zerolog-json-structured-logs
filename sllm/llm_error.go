package sllm

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/seantcanavan/zerolog-json-structured-logs/slutil"
)

// LLMError represents an error that occurred when calling an LLM provider API.
// It captures provider/model context, token usage, and the underlying error.
//
// To bridge from the OpenAI Go SDK error type, map the fields directly:
//
//	var apiErr *openai.Error
//	if errors.As(err, &apiErr) {
//	    return sllm.LogNewLLMErr(sllm.NewLLMErr{
//	        ErrorCode:  apiErr.Code,
//	        ErrorType:  apiErr.Type,
//	        InnerError: apiErr,
//	        Message:    apiErr.Message,
//	        Model:      requestModel,
//	        Provider:   "openai",
//	        RequestID:  apiErr.RequestID,
//	        StatusCode: apiErr.StatusCode,
//	    })
//	}
type LLMError struct {
	CompletionTokens int
	ErrorCode        string // provider-specific code, e.g. "rate_limit_exceeded"
	ErrorType        string // provider-specific type, e.g. "invalid_request_error"
	FinishReason     string // e.g. "stop", "length", "content_filter"
	InnerError       error
	Message          string
	Model            string // e.g. "gpt-4o", "claude-3-5-sonnet"
	PromptTokens     int
	Provider         string // e.g. "openai", "anthropic", "google"
	RequestID        string // provider request ID for tracing
	StatusCode       int
	TotalTokens      int
	Type             EnumLLMErrorType

	slutil.ExecContext `json:"execContext"`
}

// NewLLMErr is the input type for creating a LLMError. ExecContext is set automatically.
type NewLLMErr struct {
	CompletionTokens int
	ErrorCode        string
	ErrorType        string
	FinishReason     string
	InnerError       error
	Message          string
	Model            string
	PromptTokens     int
	Provider         string
	RequestID        string
	StatusCode       int
	TotalTokens      int
	Type             EnumLLMErrorType
}

// Error returns the string representation of the LLMError.
func (e *LLMError) Error() string {
	return fmt.Sprintf("[LLMError] %s/%s - %s: %v", e.Provider, e.Model, e.Message, e.InnerError)
}

// Unwrap provides the underlying error for use with errors.Is and errors.As.
func (e *LLMError) Unwrap() error {
	return e.InnerError
}

func LogNewLLMErr(newErr NewLLMErr) error {
	if newErr.Message == "" {
		newErr.Message = "a LLM error occurred"
	}

	llmErr := LLMError{
		CompletionTokens: newErr.CompletionTokens,
		ErrorCode:        newErr.ErrorCode,
		ErrorType:        newErr.ErrorType,
		ExecContext:      slutil.GetExecContext(3),
		FinishReason:     newErr.FinishReason,
		InnerError:       newErr.InnerError,
		Message:          newErr.Message,
		Model:            newErr.Model,
		PromptTokens:     newErr.PromptTokens,
		Provider:         newErr.Provider,
		RequestID:        newErr.RequestID,
		StatusCode:       newErr.StatusCode,
		TotalTokens:      newErr.TotalTokens,
		Type:             newErr.Type,
	}

	log.Error().
		Object(slutil.ZLObjectKey, &llmErr).
		Msg(newErr.Message)

	return &llmErr
}

// MarshalZerologObject allows LLMError to be logged by zerolog.
func (e *LLMError) MarshalZerologObject(zle *zerolog.Event) {
	zle.
		Int("completionTokens", e.CompletionTokens).
		Int("line", e.Line).
		Int("promptTokens", e.PromptTokens).
		Int("statusCode", e.StatusCode).
		Int("totalTokens", e.TotalTokens).
		Str("errorCode", e.ErrorCode).
		Str("errorType", e.ErrorType).
		Str("file", e.File).
		Str("finishReason", e.FinishReason).
		Str("function", e.Function).
		Str("message", e.Message).
		Str("model", e.Model).
		Str("module", e.Module).
		Str("package", e.Package).
		Str("provider", e.Provider).
		Str("requestId", e.RequestID).
		Str("type", e.Type.String())

	if e.InnerError != nil {
		zle.AnErr("innerError", e.InnerError)
	}
}

// FindOutermostLLMError returns the outermost LLMError in the error chain.
func FindOutermostLLMError(err error) *LLMError {
	res := FindLLMErrors(err)
	if len(res) > 0 {
		return res[0]
	}
	return nil
}

// FindLLMErrors returns a slice of all LLMErrors found in the error chain.
func FindLLMErrors(err error) []*LLMError {
	var errs []*LLMError
	for {
		var llmErr *LLMError
		if errors.As(err, &llmErr) {
			errs = append(errs, llmErr)
		}
		if err = errors.Unwrap(err); err == nil {
			break
		}
	}
	return errs
}
