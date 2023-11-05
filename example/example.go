package example

import (
	"errors"
	"fmt"
	sl "github.com/seantcanavan/zerolog-json-structured-logs"
	"net/http"
)

func wrapDatabaseError() error {
	newDBErr := sl.LogNewDBErr(sl.NewDBErr{ // Call LogNewDBErr to log the DB error to the temp file
		Constraint:    "pk_users",
		DBName:        "testdb",
		InternalError: errors.New("sql: no rows in result set"),
		Message:       "connection to database failed",
		Operation:     "SELECT",
		Query:         "SELECT * FROM users",
		TableName:     "users",
		Type:          sl.ErrDBConnectionFailed,
	})

	code := sl.ErrDBConnectionFailed.HTTPStatus()

	return sl.LogNewAPIErr(sl.NewAPIErr{ // call LogNewAPIErr to log the API error to the temp file
		APIEndpoint:   "/test/endpoint",
		CallerID:      "caller-123",
		InternalError: newDBErr,
		Message:       "cannot get users by address",
		RequestID:     "req-123",
		StatusCode:    code,
		UserID:        "user-123",
	})
}

// lemonadeStandError is our custom error type for the lemonade stand API.
type lemonadeStandError struct {
	Code       int    `json:"code"`
	LemonCount int    `json:"lemonCount"`
	Message    string `json:"message"`
}

// Error returns the string representation of the lemonadeStandError.
func (e lemonadeStandError) Error() string {
	return fmt.Sprintf("Error %d: %s - Lemons in stock: %d", e.Code, e.Message, e.LemonCount)
}

func wrapLibraryError() error {
	lse := lemonadeStandError{
		Code:       http.StatusTeapot,
		LemonCount: 47,
		Message:    "sorry we need 48 lemons to make lemonade",
	}

	return sl.LogNewAPIErr(sl.NewAPIErr{
		APIEndpoint:   "/lemonade/make",
		CallerID:      "caller-123",
		InternalError: lse,
		Message:       "cannot get users by address",
		RequestID:     "req-123",
		StatusCode:    http.StatusServiceUnavailable,
		UserID:        "user-123",
	})
}