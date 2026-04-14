package sldb

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/seantcanavan/zerolog-json-structured-logs/slutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

var dbLogFile *os.File // zerolog writes to this file so we can capture the output

func setupDBErrorFileLogger() {
	// have to declare this here to prevent shadowing the outer dbLogFile with :=
	var err error

	if _, err = os.Stat(slutil.TempFileNameDBLogs); err == nil {
		err = os.Remove(slutil.TempFileNameDBLogs)
		if err != nil {
			panic(fmt.Sprintf("Could not remove existing temp file: %s", err))
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		// File does not exist, which is not an error in this case,
		// but any other error accessing the file system should be reported.
		panic(fmt.Sprintf("Error checking for temp file existence: %s", err))
	}

	dbLogFile, err = os.CreateTemp("", slutil.TempFileNameDBLogs)
	if err != nil {
		panic(fmt.Sprintf("err is not nil: %s", err))
	}

	// Configure zerolog to use RFC3339Nano time for its output
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Configure zerolog to use a static now function for timestamp calculations so we can verify the timestamp later
	zerolog.TimestampFunc = slutil.StaticNowFunc

	// Configure zerolog to write to the temp file so we can easily capture the output
	log.Logger = zerolog.New(dbLogFile).With().Timestamp().Logger()
	zerolog.DisableSampling(true)
}

func tearDownDatabaseFileLogger() {
	err := os.Remove(dbLogFile.Name())
	if err != nil {
		panic(fmt.Sprintf("err is not nil: %s", err))
	}
}

func TestDBError_Error(t *testing.T) {
	expectedDBErr := DatabaseError{
		// Assume these values are what you expect to see after the operation.
		Constraint:  "Constraint",
		DBName:      "DBName",
		ExecContext: slutil.GetExecContext(3),
		InnerError:  errors.New("InnerError"),
		Message:     "Message",
		Operation:   "Operation",
		Query:       "Query",
		TableName:   "TableName",
		Type:        ErrDBConnectionFailed,
	}

	errString := expectedDBErr.Error()

	expectedString := "[DatabaseError] Operation operation on DBName.TableName with query: Query - Message - InnerError"

	assert.Equal(t, expectedString, errString)
}

func TestLogNewDBErr(t *testing.T) {
	setupDBErrorFileLogger()
	defer tearDownDatabaseFileLogger()

	// this gets propagated up to the LogItem
	message := "no users found"

	rawDBErr := DatabaseError{
		// Assume these values are what you expect to see after the operation.
		Constraint: "pk_users",
		DBName:     "testdb",
		InnerError: fmt.Errorf("wrapping error %w", errors.New("sql: no rows in result set")),
		Message:    message,
		Operation:  "SELECT",
		Query:      "SELECT * FROM users",
		TableName:  "users",
		Type:       ErrDBConnectionFailed,
	}

	newDBErr := LogNewDBErr(NewDBErr{ // Call LogNewDBErr to log the error to the temp file
		Constraint: rawDBErr.Constraint,
		DBName:     rawDBErr.DBName,
		InnerError: errors.New("sql: no rows in result set"),
		Message:    rawDBErr.Message,
		Operation:  rawDBErr.Operation,
		Query:      rawDBErr.Query,
		TableName:  rawDBErr.TableName,
		Type:       rawDBErr.Type,
	})

	// Make sure to sync and close the log file to ensure all log entries are written.
	require.NoError(t, dbLogFile.Sync())
	require.NoError(t, dbLogFile.Close())

	// Use errors.As to unwrap the error and verify that newDBErr is of type *DatabaseError
	var unwrappedNewDBErr *DatabaseError
	require.True(t, errors.As(newDBErr, &unwrappedNewDBErr), "Error is not of type *DatabaseError")

	t.Run("verify unwrappedNewDBErr has all of its fields set correctly", func(t *testing.T) {
		assert.Equal(t, rawDBErr.Constraint, unwrappedNewDBErr.Constraint)
		assert.Equal(t, rawDBErr.DBName, unwrappedNewDBErr.DBName)
		assert.True(t, strings.HasSuffix(unwrappedNewDBErr.File, "testing.go"))
		assert.Equal(t, "tRunner", unwrappedNewDBErr.Function)
		assert.Equal(t, rawDBErr.InnerError, unwrappedNewDBErr.InnerError)
		assert.NotEqual(t, rawDBErr.Line, unwrappedNewDBErr.Line) // these are called on different line numbers so should be different
		assert.Equal(t, rawDBErr.Message, unwrappedNewDBErr.Message)
		assert.Equal(t, rawDBErr.Operation, unwrappedNewDBErr.Operation)
		assert.Equal(t, rawDBErr.Query, unwrappedNewDBErr.Query)
		assert.Equal(t, rawDBErr.TableName, unwrappedNewDBErr.TableName)
		assert.Equal(t, rawDBErr.Type, unwrappedNewDBErr.Type)
		assert.EqualError(t, rawDBErr.InnerError, unwrappedNewDBErr.InnerError.Error())
	})

	t.Run("verify that jsonLogContents is well formed", func(t *testing.T) {
		// Read the log file's logFileJSONContents for assertion.
		logFileJSONContents, err := os.ReadFile(dbLogFile.Name())
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
			assert.Equal(t, unwrappedNewDBErr.Constraint, zeroLogJSONItem.ErrorAsJSON["constraint"])
			assert.Equal(t, unwrappedNewDBErr.DBName, zeroLogJSONItem.ErrorAsJSON["dbName"])
			assert.Equal(t, unwrappedNewDBErr.File, zeroLogJSONItem.ErrorAsJSON["file"])
			assert.Equal(t, unwrappedNewDBErr.Function, zeroLogJSONItem.ErrorAsJSON["function"])
			assert.Equal(t, unwrappedNewDBErr.InnerError.Error(), zeroLogJSONItem.ErrorAsJSON["innerError"]) // this is the original, top level error that DatabaseError wrapped such as a SQLError
			assert.Equal(t, float64(unwrappedNewDBErr.Line), zeroLogJSONItem.ErrorAsJSON["line"])            // you get a float64 when unmarshalling a number into interface{} for safety
			assert.Equal(t, unwrappedNewDBErr.Message, zeroLogJSONItem.ErrorAsJSON["message"])
			assert.Equal(t, unwrappedNewDBErr.Operation, zeroLogJSONItem.ErrorAsJSON["operation"])
			assert.Equal(t, unwrappedNewDBErr.Query, zeroLogJSONItem.ErrorAsJSON["query"])
			assert.Equal(t, unwrappedNewDBErr.TableName, zeroLogJSONItem.ErrorAsJSON["tableName"])

			// check for the zerolog standard values - this is critical for testing formats and outputs for things like time and level
			assert.Equal(t, zerolog.ErrorLevel.String(), zeroLogJSONItem.Level)
			assert.Equal(t, slutil.StaticNowFunc(), zeroLogJSONItem.Time)
		})

		t.Run("verify that ErrorAsJSON is well formed", func(t *testing.T) {
			dbErrEntryLogValues, ok := jsonLogContents[slutil.ZLObjectKey].(map[string]any)
			require.True(t, ok, fmt.Sprintf("%s field should be a JSON object.", slutil.ZLObjectKey))

			t.Run("verify that dbErrEntryLogValues has all of its properties and values set correctly", func(t *testing.T) {
				assert.Equal(t, unwrappedNewDBErr.Constraint, dbErrEntryLogValues["constraint"])
				assert.Equal(t, unwrappedNewDBErr.DBName, dbErrEntryLogValues["dbName"])
				assert.Equal(t, unwrappedNewDBErr.File, dbErrEntryLogValues["file"])
				assert.Equal(t, unwrappedNewDBErr.Function, dbErrEntryLogValues["function"])
				assert.Equal(t, unwrappedNewDBErr.InnerError.Error(), dbErrEntryLogValues["innerError"]) // this is the original, top level error that DatabaseError wrapped such as a SQLError
				assert.Equal(t, float64(unwrappedNewDBErr.Line), dbErrEntryLogValues["line"])            // you get a float64 when unmarshalling a number into interface{} for safety
				assert.Equal(t, unwrappedNewDBErr.Message, dbErrEntryLogValues["message"])
				assert.Equal(t, unwrappedNewDBErr.Operation, dbErrEntryLogValues["operation"])
				assert.Equal(t, unwrappedNewDBErr.Query, dbErrEntryLogValues["query"])
				assert.Equal(t, unwrappedNewDBErr.TableName, dbErrEntryLogValues["tableName"])
			})

			t.Run("verify that struct embedding is working correctly", func(t *testing.T) {
				assert.Nil(t, dbErrEntryLogValues["exec_context"]) // struct embedding means this will NOT be in the JSON
			})
		})
	})
}

func TestFindLastDatabaseError(t *testing.T) {
	firstError := LogNewDBErr(NewDBErr{
		Constraint: "pk_users_id",
		DBName:     "usersdb",
		InnerError: fmt.Errorf("primary key violation"),
		Message:    "duplicate entry for primary key",
		Operation:  "INSERT",
		Query:      "INSERT INTO users (id, name) VALUES (1, 'John Doe')",
		TableName:  "users",
		Type:       ErrDBConstraintViolated,
	})

	secondError := LogNewDBErr(NewDBErr{
		Constraint: "fk_orders_user_id",
		DBName:     "ordersdb",
		InnerError: firstError,
		Message:    "invalid foreign key",
		Operation:  "UPDATE",
		Query:      "UPDATE orders SET user_id = 2 WHERE order_id = 99",
		TableName:  "orders",
		Type:       ErrDBForeignKeyViolated,
	})

	// Test
	outermostErr := FindOutermostDatabaseError(secondError)
	require.NotNil(t, outermostErr)

	// Unwrap the error to assert on the last database error in the chain
	var outermostDBErr *DatabaseError
	require.True(t, errors.As(outermostErr, &outermostDBErr))

	var secondErrorUnwrapped *DatabaseError
	require.True(t, errors.As(outermostErr, &secondErrorUnwrapped))

	var firstErrorUnwrapped *DatabaseError
	require.True(t, errors.As(secondErrorUnwrapped.InnerError, &firstErrorUnwrapped))

	// Compare the outermost error returned to the second error defined
	assert.Equal(t, secondErrorUnwrapped.Constraint, outermostDBErr.Constraint)
	assert.Equal(t, secondErrorUnwrapped.DBName, outermostDBErr.DBName)
	assert.Equal(t, secondErrorUnwrapped.File, outermostDBErr.File)
	assert.Equal(t, secondErrorUnwrapped.Function, outermostDBErr.Function)
	assert.Equal(t, secondErrorUnwrapped.Line, outermostDBErr.Line)
	assert.Equal(t, secondErrorUnwrapped.Message, outermostDBErr.Message)
	assert.Equal(t, secondErrorUnwrapped.Operation, outermostDBErr.Operation)
	assert.Equal(t, secondErrorUnwrapped.Query, outermostDBErr.Query)
	assert.Equal(t, secondErrorUnwrapped.TableName, outermostDBErr.TableName)
	assert.Equal(t, secondErrorUnwrapped.Type, outermostDBErr.Type)

	// Compare the error wrapped by the outermost error to the first error defined
	assert.Equal(t, firstErrorUnwrapped.Constraint, firstErrorUnwrapped.Constraint)
	assert.Equal(t, firstErrorUnwrapped.DBName, firstErrorUnwrapped.DBName)
	assert.Equal(t, firstErrorUnwrapped.File, firstErrorUnwrapped.File)
	assert.Equal(t, firstErrorUnwrapped.Function, firstErrorUnwrapped.Function)
	assert.Equal(t, firstErrorUnwrapped.Line, firstErrorUnwrapped.Line)
	assert.Equal(t, firstErrorUnwrapped.Message, firstErrorUnwrapped.Message)
	assert.Equal(t, firstErrorUnwrapped.Operation, firstErrorUnwrapped.Operation)
	assert.Equal(t, firstErrorUnwrapped.Query, firstErrorUnwrapped.Query)
	assert.Equal(t, firstErrorUnwrapped.TableName, firstErrorUnwrapped.TableName)
	assert.Equal(t, firstErrorUnwrapped.Type, firstErrorUnwrapped.Type)
}
