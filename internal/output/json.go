package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// JSONResult is the standard envelope for JSON output from any lzctl command.
type JSONResult struct {
	Status string      `json:"status"`          // "ok" or "error"
	Data   interface{} `json:"data,omitempty"`  // command-specific payload
	Error  string      `json:"error,omitempty"` // error message, if any
}

// JSON writes a structured JSON result to stdout.
// Use this when --json flag is set.
func JSON(data interface{}) {
	result := JSONResult{
		Status: "ok",
		Data:   data,
	}
	writeJSON(result)
}

// JSONError writes an error result as JSON to stdout.
func JSONError(err error) {
	result := JSONResult{
		Status: "error",
		Error:  err.Error(),
	}
	writeJSON(result)
}

// JSONErrorMsg writes an error message as JSON to stdout.
func JSONErrorMsg(msg string) {
	result := JSONResult{
		Status: "error",
		Error:  msg,
	}
	writeJSON(result)
}

func writeJSON(result JSONResult) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "error encoding JSON output: %v\n", err)
	}
}
