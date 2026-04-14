package slapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/seantcanavan/zerolog-json-structured-logs/slutil"
	"net/http"
)

const CallerIDKey = "callerId"
const CallerTypeKey = "callerType"
const FileKey = "file"
const FunctionKey = "function"
const InnerErrorKey = "innerError"
const LineKey = "line"
const MessageKey = "message"
const MethodKey = "method"
const ModuleKey = "module"
const MultiParamsKey = "multiParams"
const OwnerIDKey = "ownerId"
const OwnerTypeKey = "ownerType"
const PackageKey = "package"
const PathKey = "path"
const PathParamsKey = "pathParams"
const QueryParamsKey = "queryParams"
const RequestIDKey = "requestId"
const StatusCodeKey = "statusCode"
const StatusTextKey = "statusText"

const DefaultAPIErrorMessage = "an API Error occurred"
const DefaultAPIErrorStatusCode = http.StatusInternalServerError

// APIError represents an error that occurred in the API layer of the application.
// It includes details like the HTTP status code and additional context.
type APIError struct {
	CallerID    string
	CallerType  string
	InnerError  error // An inner error if it exists such as twilio.SendSMS or other integrations
	Message     string
	Method      string
	MultiParams map[string][]string
	OwnerID     string
	OwnerType   string
	Path        string
	PathParams  map[string]string
	QueryParams map[string]string
	RequestID   string
	StatusCode  int

	slutil.ExecContext `json:"execContext"` // Embedded struct
}

// Error returns the string representation of the APIError.
func (e *APIError) Error() string {
	return fmt.Sprintf("[APIError] %d - %s at %s + %s: %s", e.StatusCode, e.Message, e.Path, e.Method, e.InnerError)
}

// Unwrap provides the underlying error for use with errors.Is and errors.As functions.
func (e *APIError) Unwrap() error {
	return e.InnerError
}

func addDefaults(apiErr *APIError) {
	if apiErr.Message == "" {
		apiErr.Message = DefaultAPIErrorMessage
	}

	if apiErr.StatusCode == 0 {
		apiErr.StatusCode = DefaultAPIErrorStatusCode
	}

	if apiErr.InnerError == nil {
		apiErr.InnerError = errors.New(apiErr.Message)
	}
}

func LogCtxInternal(ctx context.Context, err error, statusCode int) error {
	return LogCtxMsg(ctx, err, slutil.PrettyErrMsgInternal(), statusCode)
}

func LogCtxInternalF(ctx context.Context, err error, statusCode int, extra any) error {
	return LogCtxMsg(ctx, err, slutil.PrettyErrMsgInternalF(extra), statusCode)
}

func LogCtxF(ctx context.Context, err error, calleePkg, calleeFn string, statusCode int, extra any) error {
	return LogCtxMsg(ctx, err, slutil.PrettyErrMsgF(calleePkg, calleeFn, extra), statusCode)
}

func LogCtx(ctx context.Context, err error, calleePkg, calleeFn string, statusCode int) error {
	return LogCtxMsg(ctx, err, slutil.PrettyErrMsg(calleePkg, calleeFn), statusCode)
}

func LogCtxMsg(ctx context.Context, err error, message string, statusCode int) error {
	apiErr := APIError{
		CallerID:    slutil.FromCtxSafe[string](ctx, CallerIDKey),
		CallerType:  slutil.FromCtxSafe[string](ctx, CallerTypeKey),
		ExecContext: slutil.GetExecContext(3),
		InnerError:  err,
		Message:     message,
		Method:      slutil.FromCtxSafe[string](ctx, MethodKey),
		MultiParams: slutil.FromCtxSafe[map[string][]string](ctx, MultiParamsKey),
		OwnerID:     slutil.FromCtxSafe[string](ctx, OwnerIDKey),
		OwnerType:   slutil.FromCtxSafe[string](ctx, OwnerTypeKey),
		Path:        slutil.FromCtxSafe[string](ctx, PathKey),
		PathParams:  slutil.FromCtxSafe[map[string]string](ctx, PathParamsKey),
		QueryParams: slutil.FromCtxSafe[map[string]string](ctx, QueryParamsKey),
		RequestID:   slutil.FromCtxSafe[string](ctx, RequestIDKey),
		StatusCode:  statusCode,
	}

	addDefaults(&apiErr)

	log.Error().Object(slutil.ZLObjectKey, &apiErr).Send()

	return &apiErr
}

func LogNew(apiErr APIError) error {
	addDefaults(&apiErr)
	apiErr.ExecContext = slutil.GetExecContext(3)

	log.Error().Object(slutil.ZLObjectKey, &apiErr).Send()

	return &apiErr
}

func New(apiErr APIError) error {
	addDefaults(&apiErr)
	apiErr.ExecContext = slutil.GetExecContext(3)

	return &apiErr
}

// MarshalZerologObject allows APIError to be logged by zerolog.
func (e *APIError) MarshalZerologObject(zle *zerolog.Event) {
	zle.
		Int(LineKey, e.Line).
		Int(StatusCodeKey, e.StatusCode).
		Interface(MultiParamsKey, e.MultiParams).
		Interface(PathParamsKey, e.PathParams).
		Interface(QueryParamsKey, e.QueryParams).
		Str(CallerIDKey, e.CallerID).
		Str(CallerTypeKey, e.CallerType).
		Str(FileKey, e.File).
		Str(FunctionKey, e.Function).
		Str(MessageKey, e.Message).
		Str(MethodKey, e.Method).
		Str(ModuleKey, e.Module).
		Str(OwnerIDKey, e.OwnerID).
		Str(OwnerTypeKey, e.OwnerType).
		Str(PackageKey, e.Package).
		Str(PathKey, e.Path).
		Str(RequestIDKey, e.RequestID).
		Str(StatusTextKey, http.StatusText(e.StatusCode))

	if e.InnerError != nil {
		zle.AnErr(InnerErrorKey, e.InnerError)
	}
}

// FindOutermostAPIError returns the final APIError in the error chain.
func FindOutermostAPIError(err error) *APIError {
	res := FindAPIErrors(err)
	if len(res) > 0 {
		return res[0]
	}

	return nil
}

// FindAPIErrors returns a slice of all APIError found in the error chain.
func FindAPIErrors(err error) []*APIError {
	var errs []*APIError
	for {
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			errs = append(errs, apiErr)
		}
		if err = errors.Unwrap(err); err == nil {
			break
		}
	}
	return errs
}

func GenerateRandomAPIError() APIError {
	return APIError{
		CallerID:    "caller-123",
		CallerType:  "admin",
		InnerError:  fmt.Errorf("wrapping error %w", errors.New("internal server error")),
		Method:      http.MethodGet,
		MultiParams: map[string][]string{"multiKey": {"multiVal1", "multiVal2"}},
		OwnerID:     "user-123",
		OwnerType:   "user",
		Path:        "/test/endpoint",
		PathParams:  map[string]string{"pathKey1": "pathVal1", "pathKey2": "pathVal2"},
		QueryParams: map[string]string{"queryKey1": "queryVal1", "queryKey2": "queryVal2"},
		RequestID:   "req-123",
	}
}

func GenerateNonRandomAPIError() APIError {
	return APIError{
		CallerID:    "CallerID",
		CallerType:  "CallerTYpe",
		ExecContext: slutil.GetExecContext(3),
		InnerError:  errors.New("InnerError"),
		Message:     "Message",
		Method:      http.MethodGet,
		MultiParams: map[string][]string{"multiKey": {"multiVal1", "multiVal2"}},
		OwnerID:     "OwnerID",
		OwnerType:   "OwnerType",
		Path:        "Path",
		PathParams:  map[string]string{"pathKey1": "pathVal1", "pathKey2": "pathVal2"},
		QueryParams: map[string]string{"queryKey1": "queryVal1", "queryKey2": "queryVal2"},
		RequestID:   "RequestID",
		StatusCode:  500,
	}
}
