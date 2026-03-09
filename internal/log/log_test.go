package log

import (
	"testing"
)

func TestSetVerbose(t *testing.T) {
	// Default should be false.
	if Verbose() {
		t.Error("expected verbose to be false by default")
	}

	SetVerbose(true)
	if !Verbose() {
		t.Error("expected verbose to be true after SetVerbose(true)")
	}

	SetVerbose(false)
	if Verbose() {
		t.Error("expected verbose to be false after SetVerbose(false)")
	}
}

func TestDebug_NoOutput_WhenNotVerbose(t *testing.T) {
	SetVerbose(false)
	// Should not panic or produce output.
	Debug("this should not print: %s", "test")
}

func TestDebug_WhenVerbose(t *testing.T) {
	SetVerbose(true)
	defer SetVerbose(false)
	// Should not panic. Output goes to stderr which we don't capture here.
	Debug("test debug: %d", 42)
}
