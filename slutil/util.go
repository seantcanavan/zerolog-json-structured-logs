package slutil

import (
	"context"
	"fmt"
)

func FromCtxSafe[T any](ctx context.Context, key interface{}) T {
	val, ok := ctx.Value(key).(T)
	if !ok {
		// Return zero value of T if the type does not match
		return *new(T)
	}
	return val
}

func PrettyInfoMsgF(calleePkg, calleeFn string, extra any) string {
	return fmt.Sprintf("%s with result %+v", PrettyInfoMsg(calleePkg, calleeFn), extra)
}

func PrettyInfoMsg(calleePkg, calleeFn string) string {
	return fmt.Sprintf("successfully called %s.%s", calleePkg, calleeFn)
}

func PrettyErrMsg(calleePkg, calleeFn string) string {
	return fmt.Sprintf("unsuccessfully called %s.%s", calleePkg, calleeFn)
}

func PrettyErrMsgF(calleePkg, calleeFn string, extra any) string {
	return fmt.Sprintf("%s with result %+v", PrettyErrMsg(calleePkg, calleeFn), extra)
}

func PrettyErrMsgInternal() string {
	return fmt.Sprintf("encountered an error")
}

func PrettyErrMsgInternalF(extra any) string {
	return fmt.Sprintf("%s with result %+v", PrettyErrMsgInternal(), extra)
}
