//go:build !unix

package wrap

import "errors"

// Environment variables set for the wrapped shell.
const (
	EnvWrapped    = "FAMILIAR_SAYS_WRAPPED"
	EnvTranscript = "FAMILIAR_SAYS_TRANSCRIPT"
	EnvWrapPID    = "FAMILIAR_SAYS_WRAP_PID"
)

// Run is unavailable on this platform.
func Run([]string, string) (int, error) {
	return 0, errors.New("wrap needs a Unix pseudo-terminal")
}
