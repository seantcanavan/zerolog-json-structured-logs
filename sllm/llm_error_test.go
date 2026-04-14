package sllm

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/seantcanavan/zerolog-json-structured-logs/slutil"
	"github.com/seantcanavan/zerolog-json-structured-logs/slutil/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

var llmLogFile *os.File

func setupLLMErrorFileLogger() {
	var err error

	if _, err = os.Stat(testutil.TempFileNameLLMLogs); err == nil {
		err = os.Remove(testutil.TempFileNameLLMLogs)
		if err != nil {
			panic(fmt.Sprintf("Could not remove existing temp file: %s", err))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Sprintf("Error checking for temp file existence: %s", err))
	}

	llmLogFile, err = os.CreateTemp("", testutil.TempFileNameLLMLogs)
	if err != nil {
		panic(fmt.Sprintf("err is not nil: %s", err))
	}

	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFunc = testutil.StaticNowFunc
	log.Logger = zerolog.New(llmLogFile).With().Timestamp().Logger()
	zerolog.DisableSampling(true)
}

func tearDownLLMFileLogger() {
	err := os.Remove(llmLogFile.Name())
	if err != nil {
		panic(fmt.Sprintf("err is not nil: %s", err))
	}
}

func TestLLMError_Error(t *testing.T) {
	llmErr := LLMError{
		ExecContext: slutil.GetExecContext(1),
		InnerError:  errors.New("context window exceeded"),
		Message:     "prompt too long",
		Model:       "gpt-4o",
		Provider:    "openai",
	}

	expected := "[LLMError] openai/gpt-4o - prompt too long: context window exceeded"
	assert.Equal(t, expected, llmErr.Error())
}

func TestLogNewLLMErr(t *testing.T) {
	setupLLMErrorFileLogger()
	defer tearDownLLMFileLogger()

	rawLLMErr := LLMError{
		CompletionTokens: 0,
		ErrorCode:        "rate_limit_exceeded",
		ErrorType:        "invalid_request_error",
		FinishReason:     "stop",
		InnerError:       errors.New("429 Too Many Requests"),
		Message:          "rate limit hit",
		Model:            "gpt-4o",
		PromptTokens:     1000,
		Provider:         "openai",
		RequestID:        "req-abc123",
		StatusCode:       429,
		TotalTokens:      1000,
		Type:             ErrLLMRateLimitExceeded,
	}

	newLLMErr := LogNewLLMErr(NewLLMErr{
		CompletionTokens: rawLLMErr.CompletionTokens,
		ErrorCode:        rawLLMErr.ErrorCode,
		ErrorType:        rawLLMErr.ErrorType,
		FinishReason:     rawLLMErr.FinishReason,
		InnerError:       errors.New("429 Too Many Requests"),
		Message:          rawLLMErr.Message,
		Model:            rawLLMErr.Model,
		PromptTokens:     rawLLMErr.PromptTokens,
		Provider:         rawLLMErr.Provider,
		RequestID:        rawLLMErr.RequestID,
		StatusCode:       rawLLMErr.StatusCode,
		TotalTokens:      rawLLMErr.TotalTokens,
		Type:             rawLLMErr.Type,
	})

	require.NoError(t, llmLogFile.Sync())
	require.NoError(t, llmLogFile.Close())

	var unwrapped *LLMError
	require.True(t, errors.As(newLLMErr, &unwrapped), "Error is not of type *LLMError")

	t.Run("verify unwrapped has all fields set correctly", func(t *testing.T) {
		assert.Equal(t, rawLLMErr.CompletionTokens, unwrapped.CompletionTokens)
		assert.Equal(t, rawLLMErr.ErrorCode, unwrapped.ErrorCode)
		assert.Equal(t, rawLLMErr.ErrorType, unwrapped.ErrorType)
		assert.Equal(t, rawLLMErr.FinishReason, unwrapped.FinishReason)
		assert.True(t, strings.HasSuffix(unwrapped.File, "testing.go"))
		assert.Equal(t, "tRunner", unwrapped.Function)
		assert.Equal(t, rawLLMErr.Message, unwrapped.Message)
		assert.Equal(t, rawLLMErr.Model, unwrapped.Model)
		assert.NotEqual(t, rawLLMErr.Line, unwrapped.Line)
		assert.Equal(t, rawLLMErr.PromptTokens, unwrapped.PromptTokens)
		assert.Equal(t, rawLLMErr.Provider, unwrapped.Provider)
		assert.Equal(t, rawLLMErr.RequestID, unwrapped.RequestID)
		assert.Equal(t, rawLLMErr.StatusCode, unwrapped.StatusCode)
		assert.Equal(t, rawLLMErr.TotalTokens, unwrapped.TotalTokens)
		assert.Equal(t, rawLLMErr.Type, unwrapped.Type)
		assert.EqualError(t, rawLLMErr.InnerError, unwrapped.InnerError.Error())
	})

	t.Run("verify that jsonLogContents is well formed", func(t *testing.T) {
		logFileJSONContents, err := os.ReadFile(llmLogFile.Name())
		require.NoError(t, err)

		var jsonLogContents map[string]any
		require.NoError(t, json.Unmarshal(logFileJSONContents, &jsonLogContents), "Error unmarshalling log contents")
		require.NotEmpty(t, jsonLogContents)
		require.NotNil(t, jsonLogContents[slutil.ZLObjectKey], fmt.Sprintf("Log entry should contain '%s' field.", slutil.ZLObjectKey))

		t.Run("verify that jsonLogContents unmarshals into ZLJSONItem", func(t *testing.T) {
			var zeroLogJSONItem slutil.ZLJSONItem
			require.NoError(t, json.Unmarshal(logFileJSONContents, &zeroLogJSONItem))

			assert.Equal(t, float64(unwrapped.CompletionTokens), zeroLogJSONItem.ErrorAsJSON["completionTokens"])
			assert.Equal(t, unwrapped.ErrorCode, zeroLogJSONItem.ErrorAsJSON["errorCode"])
			assert.Equal(t, unwrapped.ErrorType, zeroLogJSONItem.ErrorAsJSON["errorType"])
			assert.Equal(t, unwrapped.File, zeroLogJSONItem.ErrorAsJSON["file"])
			assert.Equal(t, unwrapped.FinishReason, zeroLogJSONItem.ErrorAsJSON["finishReason"])
			assert.Equal(t, unwrapped.Function, zeroLogJSONItem.ErrorAsJSON["function"])
			assert.Equal(t, unwrapped.InnerError.Error(), zeroLogJSONItem.ErrorAsJSON["innerError"])
			assert.Equal(t, float64(unwrapped.Line), zeroLogJSONItem.ErrorAsJSON["line"])
			assert.Equal(t, unwrapped.Message, zeroLogJSONItem.ErrorAsJSON["message"])
			assert.Equal(t, unwrapped.Model, zeroLogJSONItem.ErrorAsJSON["model"])
			assert.Equal(t, float64(unwrapped.PromptTokens), zeroLogJSONItem.ErrorAsJSON["promptTokens"])
			assert.Equal(t, unwrapped.Provider, zeroLogJSONItem.ErrorAsJSON["provider"])
			assert.Equal(t, unwrapped.RequestID, zeroLogJSONItem.ErrorAsJSON["requestId"])
			assert.Equal(t, float64(unwrapped.StatusCode), zeroLogJSONItem.ErrorAsJSON["statusCode"])
			assert.Equal(t, float64(unwrapped.TotalTokens), zeroLogJSONItem.ErrorAsJSON["totalTokens"])
			assert.Equal(t, unwrapped.Type.String(), zeroLogJSONItem.ErrorAsJSON["type"])

			assert.Equal(t, zerolog.ErrorLevel.String(), zeroLogJSONItem.Level)
			assert.Equal(t, testutil.StaticNowFunc(), zeroLogJSONItem.Time)
		})

		t.Run("verify that ErrorAsJSON is well formed", func(t *testing.T) {
			llmErrEntryLogValues, ok := jsonLogContents[slutil.ZLObjectKey].(map[string]any)
			require.True(t, ok, fmt.Sprintf("%s field should be a JSON object.", slutil.ZLObjectKey))

			t.Run("verify all properties and values are set correctly", func(t *testing.T) {
				assert.Equal(t, float64(unwrapped.CompletionTokens), llmErrEntryLogValues["completionTokens"])
				assert.Equal(t, unwrapped.ErrorCode, llmErrEntryLogValues["errorCode"])
				assert.Equal(t, unwrapped.ErrorType, llmErrEntryLogValues["errorType"])
				assert.Equal(t, unwrapped.File, llmErrEntryLogValues["file"])
				assert.Equal(t, unwrapped.FinishReason, llmErrEntryLogValues["finishReason"])
				assert.Equal(t, unwrapped.Function, llmErrEntryLogValues["function"])
				assert.Equal(t, unwrapped.InnerError.Error(), llmErrEntryLogValues["innerError"])
				assert.Equal(t, float64(unwrapped.Line), llmErrEntryLogValues["line"])
				assert.Equal(t, unwrapped.Message, llmErrEntryLogValues["message"])
				assert.Equal(t, unwrapped.Model, llmErrEntryLogValues["model"])
				assert.Equal(t, float64(unwrapped.PromptTokens), llmErrEntryLogValues["promptTokens"])
				assert.Equal(t, unwrapped.Provider, llmErrEntryLogValues["provider"])
				assert.Equal(t, unwrapped.RequestID, llmErrEntryLogValues["requestId"])
				assert.Equal(t, float64(unwrapped.StatusCode), llmErrEntryLogValues["statusCode"])
				assert.Equal(t, float64(unwrapped.TotalTokens), llmErrEntryLogValues["totalTokens"])
				assert.Equal(t, unwrapped.Type.String(), llmErrEntryLogValues["type"])
			})

			t.Run("verify struct embedding is working correctly", func(t *testing.T) {
				assert.Nil(t, llmErrEntryLogValues["exec_context"])
			})
		})
	})
}

func TestFindLastLLMError(t *testing.T) {
	firstError := LogNewLLMErr(NewLLMErr{
		InnerError: errors.New("429 Too Many Requests"),
		Message:    "rate limit hit on first call",
		Model:      "gpt-4o",
		Provider:   "openai",
		Type:       ErrLLMRateLimitExceeded,
	})

	secondError := LogNewLLMErr(NewLLMErr{
		InnerError: firstError,
		Message:    "rate limit hit on retry",
		Model:      "gpt-4o",
		Provider:   "openai",
		Type:       ErrLLMRateLimitExceeded,
	})

	outermostErr := FindOutermostLLMError(secondError)
	require.NotNil(t, outermostErr)

	var secondUnwrapped *LLMError
	require.True(t, errors.As(outermostErr, &secondUnwrapped))

	var firstUnwrapped *LLMError
	require.True(t, errors.As(secondUnwrapped.InnerError, &firstUnwrapped))

	assert.Equal(t, "rate limit hit on retry", outermostErr.Message)
	assert.Equal(t, secondUnwrapped.Model, outermostErr.Model)
	assert.Equal(t, secondUnwrapped.Provider, outermostErr.Provider)
	assert.Equal(t, secondUnwrapped.Type, outermostErr.Type)
	assert.Equal(t, "rate limit hit on first call", firstUnwrapped.Message)
}
