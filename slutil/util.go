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

func UneraseMapStringArray(input map[string]any) map[string][]string {
	res := make(map[string][]string)

	for currentKey, currentVal := range input {
		var strs []string
		for _, currentAny := range currentVal.([]any) {
			strs = append(strs, currentAny.(string))
		}

		res[currentKey] = strs
	}

	return res
}

func UneraseMapString(input map[string]any) map[string]string {
	res := make(map[string]string)

	for currentKey, currentVal := range input {
		res[currentKey] = currentVal.(string)
	}

	return res
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
