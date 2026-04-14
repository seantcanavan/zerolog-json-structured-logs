package slutil

import "time"

// ZLObjectKey is the key we use to map into the .Object() call to zerolog
const ZLObjectKey = "sl" // this is our global json property key for logged items

const TempFileNameDBLogs = "testdblogs"
const TempFileNameAPILogs = "testapilogs"

type ZLJSONItem struct {
	ErrorAsJSON map[string]any `json:"sl,omitempty"`
	Level       string         `json:"level,omitempty"`
	Time        time.Time      `json:"time,omitempty"`
}
