package example

import (
	"errors"
	"fmt"
	"github.com/seantcanavan/zerolog-json-structured-logs/slapi"
	"github.com/seantcanavan/zerolog-json-structured-logs/sldb"
	"net/http"
)

func wrapDatabaseError() error {
	expectedDBErr := sldb.LogNewDBErr(sldb.NewDBErr{ // Call LogNewDBErr to log the DB error to the temp file
		Constraint: "pk_users",
		DBName:     "testdb",
		InnerError: errors.New("sql: no rows in result set"),
		Message:    "connection to database failed",
		Operation:  "SELECT",
		Query:      "SELECT * FROM users",
		TableName:  "users",
		Type:       sldb.ErrDBConnectionFailed,
	})

	apiErr := slapi.GenerateNonRandomAPIError()
	apiErr.InnerError = fmt.Errorf("wrapping db error %w", expectedDBErr)
	apiErr.StatusCode = sldb.ErrDBConnectionFailed.HTTPStatus()

	return slapi.LogNew(apiErr)
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

	apiErr := slapi.GenerateNonRandomAPIError()
	apiErr.InnerError = fmt.Errorf("wrapping db error %w", lse)
	apiErr.StatusCode = http.StatusServiceUnavailable

	return slapi.LogNew(apiErr)
}
