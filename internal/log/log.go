package log

import (
	"fmt"
	"os"

	"github.com/jwalton/gchalk"
)

// LogError writes an error message to stderr.
func LogError(message interface{}) {
	LogErrorf("%v", message)
}

// LogErrorf writes a formatted error message to stderr.
func LogErrorf(message string, a ...interface{}) {
	os.Stderr.Write([]byte(gchalk.Stderr.BrightRed(fmt.Sprintf(message, a...)) + "\n"))
}

// LogFatal writes an error message to stderr, and then exits with a non-zero status code.
func LogFatal(message interface{}) {
	LogFatalf("%v", message)
}

// LogFatalf writes a  formatted error message to stderr, and then exits with a non-zero status code.
func LogFatalf(message string, a ...interface{}) {
	LogErrorf(message, a...)
	os.Exit(1)
}
