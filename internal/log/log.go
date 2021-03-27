package log

import (
	"fmt"
	"os"

	"github.com/jwalton/gchalk"
)

// PixdlError writes an error message to stderr.
func PixdlError(message interface{}) {
	PixdlErrorf("%v", message)
}

// PixdlErrorf writes a formatted error message to stderr.
func PixdlErrorf(message string, a ...interface{}) {
	os.Stderr.Write([]byte(gchalk.Stderr.BrightRed(fmt.Sprintf(message, a...)) + "\n"))
}

// PixdlFatal writes an error message to stderr, and then exits with a non-zero status code.
func PixdlFatal(message interface{}) {
	PixdlFatalf("%v", message)
}

// PixdlFatalf writes a  formatted error message to stderr, and then exits with a non-zero status code.
func PixdlFatalf(message string, a ...interface{}) {
	PixdlErrorf(message, a...)
	os.Exit(1)
}
