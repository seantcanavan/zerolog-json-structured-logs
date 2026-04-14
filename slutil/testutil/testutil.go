package testutil

import (
	"math/rand"
	"time"
)

const TempFileNameAPILogs = "testapilogs"
const TempFileNameDBLogs = "testdblogs"
const TempFileNameLLMLogs = "testllmlogs"

// StaticNow generates a single random time on package initialization for use as a deterministic
// timestamp in tests. Using a random date each run helps surface date-handling edge cases.
var StaticNow = func() time.Time {
	nowRand := rand.New(rand.NewSource(time.Now().Unix()))
	year := nowRand.Intn(2023-2000) + 2000
	month := time.Month(nowRand.Intn(12) + 1)
	day := nowRand.Intn(28) + 1
	hour := nowRand.Intn(24)
	minute := nowRand.Intn(60)
	second := nowRand.Intn(60)
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
}()

func StaticNowFunc() time.Time {
	return StaticNow
}

// UneraseMapStringArray converts a map[string]any (from JSON unmarshalling) back to map[string][]string.
func UneraseMapStringArray(input map[string]any) map[string][]string {
	res := make(map[string][]string)
	for k, v := range input {
		var strs []string
		for _, item := range v.([]any) {
			strs = append(strs, item.(string))
		}
		res[k] = strs
	}
	return res
}

// UneraseMapString converts a map[string]any (from JSON unmarshalling) back to map[string]string.
func UneraseMapString(input map[string]any) map[string]string {
	res := make(map[string]string)
	for k, v := range input {
		res[k] = v.(string)
	}
	return res
}
