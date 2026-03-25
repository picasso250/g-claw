package gateway

import "io"

var debugLogWriter io.Writer

func SetDebugLogWriter(w io.Writer) {
	debugLogWriter = w
}

func DebugMirrorWriter(defaultWriter io.Writer) io.Writer {
	if debugLogWriter == nil {
		return defaultWriter
	}
	return io.MultiWriter(defaultWriter, debugLogWriter)
}
