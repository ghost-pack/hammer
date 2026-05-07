package logging

import "runtime"

func frame(pc uintptr) runtime.Frame {
	frames := runtime.CallersFrames([]uintptr{pc})
	f, _ := frames.Next()
	return f
}
