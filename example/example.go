package example

import (
	"errors"
	"fmt"
	"github.com/seantcanavan/zerolog-json-structured-logs/slapi"
	"github.com/seantcanavan/zerolog-json-structured-logs/sldb"
	"net/http"
)

func wrapDatabaseError() error {
	expectedDBErr := sldb.LogNewDBErr(sldb.NewDBErr{
		Constraint: "pk_users",
		DBName:     "testdb",
		InnerError: errors.New("sql: no rows in result set"),
		Message:    "connection to database failed",
		Operation:  "SELECT",
		Query:      "SELECT * FROM users",
		TableName:  "users",
		Type:       sldb.ErrDBConnectionFailed,
	})

	apiErr := slapi.APIError{
		CallerID:    "CallerID",
		CallerType:  "CallerTYpe",
		Method:      http.MethodGet,
		MultiParams: map[string][]string{"multiKey": {"multiVal1", "multiVal2"}},
		OwnerID:     "OwnerID",
		OwnerType:   "OwnerType",
		Path:        "Path",
		PathParams:  map[string]string{"pathKey1": "pathVal1", "pathKey2": "pathVal2"},
		QueryParams: map[string]string{"queryKey1": "queryVal1", "queryKey2": "queryVal2"},
		RequestID:   "RequestID",
	}
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

	apiErr := slapi.APIError{
		CallerID:    "CallerID",
		CallerType:  "CallerTYpe",
		Method:      http.MethodGet,
		MultiParams: map[string][]string{"multiKey": {"multiVal1", "multiVal2"}},
		OwnerID:     "OwnerID",
		OwnerType:   "OwnerType",
		Path:        "Path",
		PathParams:  map[string]string{"pathKey1": "pathVal1", "pathKey2": "pathVal2"},
		QueryParams: map[string]string{"queryKey1": "queryVal1", "queryKey2": "queryVal2"},
		RequestID:   "RequestID",
	}
	apiErr.InnerError = fmt.Errorf("wrapping db error %w", lse)
	apiErr.StatusCode = http.StatusServiceUnavailable

	return slapi.LogNew(apiErr)
}
