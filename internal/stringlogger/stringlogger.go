package stringlogger

import "strings"

const DefaultSeparator = "\n"

type StringLogger struct {
	buffer    strings.Builder
	separator string
	callback  func(string)
}

func (bl *StringLogger) Write(bs []byte) (bytesRead int, err error) {
	bytesRead = len(bs)
	bsStr := string(bs)

	// No separator => store all the contents in buffer
	if !strings.Contains(bsStr, bl.separator) {
		bl.buffer.WriteString(bsStr)
		return
	}

	// Split input by separator
	lines := strings.Split(bsStr, bl.separator)

	// Buffer has content => concat with first line and log it
	if bl.buffer.Len() > 0 {
		bl.callback(bl.buffer.String() + lines[0])
		bl.buffer.Reset()
	} else {
		bl.callback(lines[0])
	}

	// Write all full lines to logger
	for i := 1; i < len(lines)-1; i += 1 {
		bl.callback(lines[i])
	}

	// Buffer last line
	bl.buffer.WriteString(lines[len(lines)-1])

	return
}

// Close logs the remaining buffer
func (bl *StringLogger) Close() {
	bl.callback(bl.buffer.String())
	bl.buffer.Reset()
}

func NewWithSeparator(separator string, callback func(string)) *StringLogger {
	return &StringLogger{
		separator: separator,
		callback: callback,
	}
}

func New(callback func(string)) *StringLogger {
	return NewWithSeparator(DefaultSeparator, callback)
}
