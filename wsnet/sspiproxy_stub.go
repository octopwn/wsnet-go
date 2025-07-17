// sspiproxy_stub.go
//go:build !windows
// +build !windows

package wsnet

// This file provides no-op stubs for SSPI on non-Windows platforms so the
// rest of the codebase can compile while clearly indicating unsupported
// functionality.

import (
    "errors"
    "fmt"
)

// --- Constants / Types (trimmed to what clienthandler expects) ---

type ContextAttributes uint32

// SSPISession is a stub that returns errors for all operations.
// Having the type defined lets ClientHandler build on every platform.

type SSPISession struct{}

func NewSSPISession() *SSPISession { return &SSPISession{} }

var ErrNotSupported = errors.New("SSPI functionality is not supported on this platform")

func (s *SSPISession) NtlmAuth(username string, credUsage int, targetName string, contextAttrs int) (bool, []byte, error) {
    return false, nil, fmt.Errorf("ntlmauth not supported: %w", ErrNotSupported)
}
func (s *SSPISession) NtlmChallenge(contextAttrs int, authData []byte, targetName string) (bool, []byte, error) {
    return false, nil, fmt.Errorf("ntlmauth challenge not supported: %w", ErrNotSupported)
}
func (s *SSPISession) Kerberos(username string, credUsage int, targetName string, contextAttrs int, authData []byte) (bool, []byte, error) {
    return false, nil, fmt.Errorf("kerberos not supported: %w", ErrNotSupported)
}
func (s *SSPISession) SessionKey() (bool, []byte, error) {
    return false, nil, fmt.Errorf("session key not supported: %w", ErrNotSupported)
}
func (s *SSPISession) SequenceNo() (bool, []byte, error) {
    return false, nil, fmt.Errorf("sequence no not supported: %w", ErrNotSupported)
}
func (s *SSPISession) processMessage(message WSPacket) (*WSPacket, error) {
    return nil, fmt.Errorf("sspiproxy not supported on this platform: %w", ErrNotSupported)
} 