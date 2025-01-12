// sspiproxy_linux.go
//go:build !windows
// +build !windows

package wsnet

// This whole file is only here to mirror the Windows implementation.
// The actual implementation is not supported on Linux.

import (
	"errors"
	"fmt"
)

// --- Enums / Constants ---

// SEC_E represents security error codes.
type SEC_E uint32

const (
	SEC_E_OK                          SEC_E = 0x00000000
	SEC_E_CONTINUE_NEEDED             SEC_E = 0x00090312
	SEC_E_INSUFFICIENT_MEMORY         SEC_E = 0x80090300
	SEC_E_INTERNAL_ERROR              SEC_E = 0x80090304
	SEC_E_INVALID_HANDLE              SEC_E = 0x80090301
	SEC_E_INVALID_TOKEN               SEC_E = 0x80090308
	SEC_E_LOGON_DENIED                SEC_E = 0x8009030C
	SEC_E_NO_AUTHENTICATING_AUTHORITY SEC_E = 0x80090311
	SEC_E_NO_CREDENTIALS              SEC_E = 0x8009030E
	SEC_E_TARGET_UNKNOWN              SEC_E = 0x80090303
	SEC_E_UNSUPPORTED_FUNCTION        SEC_E = 0x80090302
	SEC_E_WRONG_PRINCIPAL             SEC_E = 0x80090322
	SEC_E_NOT_OWNER                   SEC_E = 0x80090306
	SEC_E_SECPKG_NOT_FOUND            SEC_E = 0x80090305
	SEC_E_UNKNOWN_CREDENTIALS         SEC_E = 0x8009030D
	SEC_E_RENEGOTIATE                 SEC_E = 590625
	SEC_E_COMPLETE_AND_CONTINUE       SEC_E = 590612
	SEC_E_COMPLETE_NEEDED             SEC_E = 590611
	SEC_E_INCOMPLETE_CREDENTIALS      SEC_E = 590624
)

// SecBufferType represents the type of security buffer.
type SecBufferType int

const (
	SECBUFFER_VERSION SecBufferType = 0
	SECBUFFER_EMPTY   SecBufferType = 0
	SECBUFFER_DATA    SecBufferType = 1
	SECBUFFER_TOKEN   SecBufferType = 2
)

// ContextAttributes represents the context attribute flags.
type ContextAttributes uint32

// Define necessary context attributes as needed.
const (
	ISC_REQ_CONFIDENTIALITY ContextAttributes = 0x00000010
	ISC_REQ_MUTUAL_AUTH     ContextAttributes = 0x00000002
	ISC_REQ_DELEGATE        ContextAttributes = 0x00000001
	// Add more as required.
)

// --- Structs ---

// SecHandle mimics the C# SecHandle struct.
type SecHandle struct {
	DwLower uintptr
	DwUpper uintptr
}

// SecBuffer mimics the C# SecBuffer struct.
type SecBuffer struct {
	CbBuffer   int
	BufferType int
	PvBuffer   []byte
}

// SecBufferDesc mimics the C# SecBufferDesc struct.
type SecBufferDesc struct {
	UlVersion int
	CBuffers  int
	PBuffers  []SecBuffer
}

// SECURITY_INTEGER mimics the C# SECURITY_INTEGER struct.
type SECURITY_INTEGER struct {
	LowPart  uint32
	HighPart int32
}

// SECURITY_HANDLE mimics the C# SECURITY_HANDLE struct.
type SECURITY_HANDLE struct {
	LowPart  uintptr
	HighPart uintptr
}

// SecPkgContext_Sizes mimics the C# SecPkgContext_Sizes struct.
type SecPkgContext_Sizes struct {
	CbMaxToken        uint32
	CbMaxSignature    uint32
	CbBlockSize       uint32
	CbSecurityTrailer uint32
}

// SecPkgContext_SessionKey mimics the C# SecPkgContext_SessionKey struct.
type SecPkgContext_SessionKey struct {
	SessionKeyLength uint32
	SessionKey       []byte
}

// --- SSPISession Struct ---

// SSPISession mimics the C# SSPISession class.
type SSPISession struct {
	// Fields are included to match the C# class, but are unused in this stub.
	phCredential   SECURITY_HANDLE
	hClientContext SECURITY_HANDLE
	ptsExpiry      SECURITY_INTEGER
	ContextAttrs   ContextAttributes
	TargetDataRep  int
	// Additional fields can be added if needed.
}

// --- Constructor ---

// NewSSPISession creates a new instance of SSPISession.
func NewSSPISession() *SSPISession {
	return &SSPISession{
		phCredential:   SECURITY_HANDLE{},
		hClientContext: SECURITY_HANDLE{},
		ptsExpiry:      SECURITY_INTEGER{},
		ContextAttrs:   0,
		TargetDataRep:  0,
	}
}

// --- Error Definitions ---

var ErrNotSupported = errors.New("SSPI functionality is not supported on this platform")

// --- SSPISession Methods ---

// ntlmauth mimics the C# ntlmauth method.
func (s *SSPISession) NtlmAuth(username string, credUsage int, targetName string, contextAttrs int) (bool, []byte, error) {
	return false, nil, fmt.Errorf("ntlmauth not supported on Linux: %w", ErrNotSupported)
}

// ntlmchallenge mimics the C# ntlmchallenge method.
func (s *SSPISession) NtlmChallenge(contextAttrs int, authData []byte, targetName string) (bool, []byte, error) {
	return false, nil, fmt.Errorf("ntlmchallenge not supported on Linux: %w", ErrNotSupported)
}

// kerberos mimics the C# kerberos method.
func (s *SSPISession) Kerberos(username string, credUsage int, targetName string, contextAttrs int, authData []byte) (bool, []byte, error) {
	return false, nil, fmt.Errorf("kerberos authentication not supported on Linux: %w", ErrNotSupported)
}

// sessionkey mimics the C# sessionkey method.
func (s *SSPISession) SessionKey() (bool, []byte, error) {
	return false, nil, fmt.Errorf("sessionkey retrieval not supported on Linux: %w", ErrNotSupported)
}

// sequenceno mimics the C# sequenceno method.
func (s *SSPISession) SequenceNo() (bool, []byte, error) {
	return false, nil, fmt.Errorf("sequenceno retrieval not supported on Linux: %w", ErrNotSupported)
}

func (s *SSPISession) processMessage(message WSPacket) (*WSPacket, error) {
	return nil, fmt.Errorf("processMessage not supported on Linux: %w", ErrNotSupported)
}

// --- Additional Helper Types and Methods ---

// CMDAuthErr mimics a C# error type used in the original code.
// This is a placeholder. Adjust as necessary based on your actual implementation.
type CMDAuthErr struct {
	Code    int
	Message string
}

// toBytes mimics the C# to_bytes method for CMDAuthErr.
func (e *CMDAuthErr) ToBytes() []byte {
	// Placeholder implementation. Adjust serialization as needed.
	return []byte(fmt.Sprintf("Error %d: %s", e.Code, e.Message))
}

// CMDNTLMAuthReply mimics a C# reply type used in the original code.
// This is a placeholder. Adjust as necessary based on your actual implementation.
type CMDNTLMAuthReply struct {
	Code         int
	ContextAttrs int
	AuthData     []byte
}

// toBytes mimics the C# to_bytes method for CMDNTLMAuthReply.
func (r *CMDNTLMAuthReply) ToBytes() []byte {
	// Placeholder implementation. Adjust serialization as needed.
	return []byte(fmt.Sprintf("NTLMAuthReply Code: %d, ContextAttrs: %d, AuthData Length: %d", r.Code, r.ContextAttrs, len(r.AuthData)))
}

// CMDNTLMChallengeReply mimics a C# reply type used in the original code.
// This is a placeholder. Adjust as necessary based on your actual implementation.
type CMDNTLMChallengeReply struct {
	Code         int
	ContextAttrs int
	AuthData     []byte
}

// toBytes mimics the C# to_bytes method for CMDNTLMChallengeReply.
func (r *CMDNTLMChallengeReply) ToBytes() []byte {
	// Placeholder implementation. Adjust serialization as needed.
	return []byte(fmt.Sprintf("NTLMChallengeReply Code: %d, ContextAttrs: %d, AuthData Length: %d", r.Code, r.ContextAttrs, len(r.AuthData)))
}

// CMDKerberosReply mimics a C# reply type used in the original code.
// This is a placeholder. Adjust as necessary based on your actual implementation.
type CMDKerberosReply struct {
	Code         int
	ContextAttrs int
	AuthData     []byte
}

// toBytes mimics the C# to_bytes method for CMDKerberosReply.
func (r *CMDKerberosReply) ToBytes() []byte {
	// Placeholder implementation. Adjust serialization as needed.
	return []byte(fmt.Sprintf("KerberosReply Code: %d, ContextAttrs: %d, AuthData Length: %d", r.Code, r.ContextAttrs, len(r.AuthData)))
}

// CMDSessionKeyReply mimics a C# reply type used in the original code.
// This is a placeholder. Adjust as necessary based on your actual implementation.
type CMDSessionKeyReply struct {
	Code       int
	SessionKey []byte
}

// toBytes mimics the C# to_bytes method for CMDSessionKeyReply.
func (r *CMDSessionKeyReply) ToBytes() []byte {
	// Placeholder implementation. Adjust serialization as needed.
	return []byte(fmt.Sprintf("SessionKeyReply Code: %d, SessionKey Length: %d", r.Code, len(r.SessionKey)))
}

// CMDSequenceReply mimics a C# reply type used in the original code.
// This is a placeholder. Adjust as necessary based on your actual implementation.
type CMDSequenceReply struct {
	Code    int
	EncData []byte
}

// toBytes mimics the C# to_bytes method for CMDSequenceReply.
func (r *CMDSequenceReply) ToBytes() []byte {
	// Placeholder implementation. Adjust serialization as needed.
	return []byte(fmt.Sprintf("SequenceReply Code: %d, EncData Length: %d", r.Code, len(r.EncData)))
}

func localSSPITest() {
	// Placeholder function to test the local implementation.
	// This is a placeholder. Adjust as necessary based on your actual implementation.
	fmt.Println("Local SSPI test function")
}
