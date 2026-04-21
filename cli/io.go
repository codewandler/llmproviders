// Package cli provides importable CLI commands for llmproviders.
// These commands can be embedded in other CLIs (like miniagent) or used standalone.
package cli

import (
	"io"
	"os"
)

// IO provides abstracted I/O streams for CLI commands.
// This allows embedding CLIs to control input/output for testing and composition.
type IO struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

// WithDefaults returns a copy of IO with nil streams replaced by os.Std* defaults.
func (cfg IO) WithDefaults() IO {
	if cfg.In == nil {
		cfg.In = os.Stdin
	}
	if cfg.Out == nil {
		cfg.Out = os.Stdout
	}
	if cfg.Err == nil {
		cfg.Err = os.Stderr
	}
	return cfg
}

// DefaultIO returns an IO configured with standard streams.
func DefaultIO() IO {
	return IO{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}
}
