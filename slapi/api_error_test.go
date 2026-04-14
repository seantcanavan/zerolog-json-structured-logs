package slapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/seantcanavan/zerolog-json-structured-logs/slutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

var apiLogFile *os.File // zerolog writes to this file so we can capture the output

func setupAPIErrorFileLogger() {
	// have to declare this here to prevent shadowing the outer apiLogFile with :=
	var err error

	if _, err = os.Stat(slutil.TempFileNameAPILogs); err == nil {
		err = os.Remove(slutil.TempFileNameAPILogs)
		if err != nil {
			panic(fmt.Sprintf("Could not remove existing temp file: %s", err))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		// File does not exist, which is not an error in this case,
		// but any other error accessing the file system should be reported.
		panic(fmt.Sprintf("Error checking for temp file existence: %s", err))
	}

	apiLogFile, err = os.CreateTemp("", slutil.TempFileNameAPILogs)
	if err != nil {
		panic(fmt.Sprintf("err is not nil: %s", err))
	}

	// Configure zerolog to use RFC3339Nano time for its output
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Configure zerolog to use a static now function for timestamp calculations so we can verify the timestamp later
	zerolog.TimestampFunc = slutil.StaticNowFunc

	// Configure zerolog to write to the temp file so we can easily capture the output
	log.Logger = zerolog.New(apiLogFile).With().Timestamp().Logger()
	zerolog.DisableSampling(true)
}

func tearDownAPIFileLogger() {
	err := os.Remove(apiLogFile.Name())
	if err != nil {
		panic(fmt.Sprintf("err is not nil: %s", err))
	}
}

func TestAPIError_Error(t *testing.T) {
	// Set up fake values for the expected API error
	expectedAPIError := APIError{
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

	// Define the expected string output from the Error() method
	expectedString := "[APIError] 500 - Message at Path + GET: InnerError"

	// Get the actual error string from the APIError instance
	errString := expectedAPIError.Error()

	// Assert that the expected string matches the actual error string
	assert.Equal(t, expectedString, errString)
}

func TestLogCtxMsg(t *testing.T) {
	setupAPIErrorFileLogger()
	defer tearDownAPIFileLogger()

	ctx := context.Background()

	rawAPIError := GenerateRandomAPIError()

	ctx = context.WithValue(ctx, CallerIDKey, rawAPIError.CallerID)
	ctx = context.WithValue(ctx, CallerTypeKey, rawAPIError.CallerType)
	ctx = context.WithValue(ctx, MethodKey, rawAPIError.Method)
	ctx = context.WithValue(ctx, MultiParamsKey, rawAPIError.MultiParams)
	ctx = context.WithValue(ctx, OwnerIDKey, rawAPIError.OwnerID)
	ctx = context.WithValue(ctx, OwnerTypeKey, rawAPIError.OwnerType)
	ctx = context.WithValue(ctx, PathKey, rawAPIError.Path)
	ctx = context.WithValue(ctx, PathParamsKey, rawAPIError.PathParams)
	ctx = context.WithValue(ctx, QueryParamsKey, rawAPIError.QueryParams)
	ctx = context.WithValue(ctx, RequestIDKey, rawAPIError.RequestID)

	loggedAPIError := LogCtxMsg(ctx, rawAPIError.InnerError, "", 0)

	// Make sure to sync and close the log file to ensure all log entries are written.
	require.NoError(t, apiLogFile.Sync())
	require.NoError(t, apiLogFile.Close())

	verifyAPILogContents(t, &rawAPIError, loggedAPIError)
}

func TestLogNew(t *testing.T) {
	setupAPIErrorFileLogger()
	defer tearDownAPIFileLogger()

	rawAPIError := GenerateRandomAPIError()

	loggedAPIError := LogNew(rawAPIError)

	// Make sure to sync and close the log file to ensure all log entries are written.
	require.NoError(t, apiLogFile.Sync())
	require.NoError(t, apiLogFile.Close())

	verifyAPILogContents(t, &rawAPIError, loggedAPIError)
}

func verifyAPILogContents(t *testing.T, rawAPIError *APIError, loggedAPIError error) {
	// Use errors.As to unwrap the error and verify that loggedAPIError is of type *APIError
	var unwrappedAPIErr *APIError
	require.True(t, errors.As(loggedAPIError, &unwrappedAPIErr), "Error is not of type *APIError")

	t.Run("verify unwrappedAPIErr has all of its fields set correctly", func(t *testing.T) {
		assert.NotEqual(t, rawAPIError.Line, unwrappedAPIErr.Line) // these are called on different line numbers so should be different
		assert.Equal(t, DefaultAPIErrorStatusCode, unwrappedAPIErr.StatusCode)

		assert.Equal(t, rawAPIError.MultiParams, unwrappedAPIErr.MultiParams)
		assert.Equal(t, rawAPIError.PathParams, unwrappedAPIErr.PathParams)
		assert.Equal(t, rawAPIError.QueryParams, unwrappedAPIErr.QueryParams)

		assert.Equal(t, rawAPIError.CallerID, unwrappedAPIErr.CallerID)
		assert.Equal(t, rawAPIError.CallerType, unwrappedAPIErr.CallerType)
		assert.Equal(t, "tRunner", unwrappedAPIErr.Function)
		assert.True(t, strings.HasSuffix(unwrappedAPIErr.File, "testing.go"))
		assert.Equal(t, "testing", unwrappedAPIErr.Package)
		assert.Equal(t, "", unwrappedAPIErr.Module)
		assert.Equal(t, DefaultAPIErrorMessage, unwrappedAPIErr.Message)
		assert.Equal(t, rawAPIError.Method, unwrappedAPIErr.Method)
		assert.Equal(t, rawAPIError.OwnerID, unwrappedAPIErr.OwnerID)
		assert.Equal(t, rawAPIError.OwnerType, unwrappedAPIErr.OwnerType)
		assert.Equal(t, rawAPIError.Path, unwrappedAPIErr.Path)
		assert.Equal(t, rawAPIError.RequestID, unwrappedAPIErr.RequestID)

		assert.Equal(t, rawAPIError.InnerError, unwrappedAPIErr.InnerError)
		assert.EqualError(t, rawAPIError.InnerError, unwrappedAPIErr.InnerError.Error())
	})

	t.Run("verify that jsonLogContents is well formed", func(t *testing.T) {
		// Read the log file's logFileJSONContents for assertion.
		logFileJSONContents, err := os.ReadFile(apiLogFile.Name())
		require.NoError(t, err)

		// Unmarshal logFileJSONContents into a generic map[string]any
		var jsonLogContents map[string]any
		require.NoError(t, json.Unmarshal(logFileJSONContents, &jsonLogContents), "Error unmarshalling log logFileJSONContents")
		require.NotEmpty(t, jsonLogContents, "Log file should contain at least one entry.")
		require.NotNil(t, jsonLogContents[slutil.ZLObjectKey], fmt.Sprintf("Log entry should contain '%s' field.", slutil.ZLObjectKey))

		t.Run("verify that jsonLogContents unmarshals into an instance of ZLJSONItem", func(t *testing.T) {
			var zeroLogJSONItem slutil.ZLJSONItem
			require.NoError(t, json.Unmarshal(logFileJSONContents, &zeroLogJSONItem), "json.Unmarshal should not have produced an error")

			// check for the error values embedded in the top-level logging struct
			assert.Equal(t, float64(unwrappedAPIErr.Line), zeroLogJSONItem.ErrorAsJSON[LineKey]) // you get a float64 when unmarshalling a number into interface{} for safety
			assert.Equal(t, float64(unwrappedAPIErr.StatusCode), zeroLogJSONItem.ErrorAsJSON[StatusCodeKey])

			assert.Equal(t, unwrappedAPIErr.MultiParams, slutil.UneraseMapStringArray(zeroLogJSONItem.ErrorAsJSON[MultiParamsKey].(map[string]any)))
			assert.Equal(t, unwrappedAPIErr.PathParams, slutil.UneraseMapString(zeroLogJSONItem.ErrorAsJSON[PathParamsKey].(map[string]any)))
			assert.Equal(t, unwrappedAPIErr.QueryParams, slutil.UneraseMapString(zeroLogJSONItem.ErrorAsJSON[QueryParamsKey].(map[string]any)))

			assert.Equal(t, unwrappedAPIErr.CallerID, zeroLogJSONItem.ErrorAsJSON[CallerIDKey])
			assert.Equal(t, unwrappedAPIErr.CallerType, zeroLogJSONItem.ErrorAsJSON[CallerTypeKey])
			assert.Equal(t, unwrappedAPIErr.File, zeroLogJSONItem.ErrorAsJSON[FileKey])
			assert.Equal(t, unwrappedAPIErr.Function, zeroLogJSONItem.ErrorAsJSON[FunctionKey])
			assert.Equal(t, unwrappedAPIErr.Message, zeroLogJSONItem.ErrorAsJSON[MessageKey])
			assert.Equal(t, unwrappedAPIErr.Message, DefaultAPIErrorMessage)
			assert.Equal(t, unwrappedAPIErr.Method, zeroLogJSONItem.ErrorAsJSON[MethodKey])
			assert.Equal(t, unwrappedAPIErr.Module, zeroLogJSONItem.ErrorAsJSON[ModuleKey])
			assert.Equal(t, unwrappedAPIErr.OwnerID, zeroLogJSONItem.ErrorAsJSON[OwnerIDKey])
			assert.Equal(t, unwrappedAPIErr.OwnerType, zeroLogJSONItem.ErrorAsJSON[OwnerTypeKey])
			assert.Equal(t, unwrappedAPIErr.Package, zeroLogJSONItem.ErrorAsJSON[PackageKey])
			assert.Equal(t, unwrappedAPIErr.Path, zeroLogJSONItem.ErrorAsJSON[PathKey])
			assert.Equal(t, unwrappedAPIErr.RequestID, zeroLogJSONItem.ErrorAsJSON[RequestIDKey])

			assert.Equal(t, http.StatusText(unwrappedAPIErr.StatusCode), zeroLogJSONItem.ErrorAsJSON[StatusTextKey])
			assert.Equal(t, unwrappedAPIErr.InnerError.Error(), zeroLogJSONItem.ErrorAsJSON[InnerErrorKey]) // this is the original, top level error that DatabaseError wrapped such as a SQLError

			// check for the zerolog standard values - this is critical for testing formats and outputs for things like time and level
			assert.Equal(t, zerolog.ErrorLevel.String(), zeroLogJSONItem.Level)
			assert.Equal(t, slutil.StaticNowFunc(), zeroLogJSONItem.Time)
		})

		t.Run("verify that ErrorAsJSON is well formed", func(t *testing.T) {
			apiErrEntryLogValues, ok := jsonLogContents[slutil.ZLObjectKey].(map[string]any)
			require.True(t, ok, fmt.Sprintf("%s field should be a JSON object.", slutil.ZLObjectKey))

			t.Run("verify that apiErrEntryLogValues has all of its properties and values set correctly", func(t *testing.T) {
				assert.Equal(t, float64(unwrappedAPIErr.Line), apiErrEntryLogValues[LineKey]) // you get a float64 when unmarshalling a number into interface{} for safety
				assert.Equal(t, float64(unwrappedAPIErr.StatusCode), apiErrEntryLogValues[StatusCodeKey])

				assert.Equal(t, unwrappedAPIErr.MultiParams, slutil.UneraseMapStringArray(apiErrEntryLogValues[MultiParamsKey].(map[string]any)))
				assert.Equal(t, unwrappedAPIErr.PathParams, slutil.UneraseMapString(apiErrEntryLogValues[PathParamsKey].(map[string]any)))
				assert.Equal(t, unwrappedAPIErr.QueryParams, slutil.UneraseMapString(apiErrEntryLogValues[QueryParamsKey].(map[string]any)))

				assert.Equal(t, unwrappedAPIErr.CallerID, apiErrEntryLogValues[CallerIDKey])
				assert.Equal(t, unwrappedAPIErr.CallerType, apiErrEntryLogValues[CallerTypeKey])
				assert.Equal(t, unwrappedAPIErr.File, apiErrEntryLogValues[FileKey])
				assert.Equal(t, unwrappedAPIErr.Module, apiErrEntryLogValues[ModuleKey])
				assert.Equal(t, unwrappedAPIErr.Function, apiErrEntryLogValues[FunctionKey])
				assert.Equal(t, unwrappedAPIErr.Message, apiErrEntryLogValues[MessageKey])
				assert.Equal(t, unwrappedAPIErr.Method, apiErrEntryLogValues[MethodKey])
				assert.Equal(t, unwrappedAPIErr.OwnerID, apiErrEntryLogValues[OwnerIDKey])
				assert.Equal(t, unwrappedAPIErr.OwnerType, apiErrEntryLogValues[OwnerTypeKey])
				assert.Equal(t, unwrappedAPIErr.Package, apiErrEntryLogValues[PackageKey])
				assert.Equal(t, unwrappedAPIErr.Path, apiErrEntryLogValues[PathKey])
				assert.Equal(t, unwrappedAPIErr.RequestID, apiErrEntryLogValues[RequestIDKey])

				assert.Equal(t, http.StatusText(unwrappedAPIErr.StatusCode), apiErrEntryLogValues[StatusTextKey])
				assert.Equal(t, unwrappedAPIErr.InnerError.Error(), apiErrEntryLogValues[InnerErrorKey]) // this is the original, top level error that DatabaseError wrapped such as a SQLError
			})

			t.Run("verify that struct embedding is working correctly", func(t *testing.T) {
				assert.Nil(t, apiErrEntryLogValues["exec_context"]) // struct embedding means this will NOT be in the JSON
			})
		})
	})
}
