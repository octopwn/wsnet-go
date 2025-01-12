// sspiproxy_windows.go
//go:build windows
// +build windows

package wsnet

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
	//"golang.org/x/sys/windows"
)

// --- Enums / Constants ---

// Common SSPI result codes (SEC_E, etc.) – not exhaustive:
const (
	SEC_E_OK                          = 0x00000000
	SEC_E_CONTINUE_NEEDED             = 0x00090312
	SEC_E_INSUFFICIENT_MEMORY         = 0x80090300
	SEC_E_INTERNAL_ERROR              = 0x80090304
	SEC_E_INVALID_HANDLE              = 0x80090301
	SEC_E_INVALID_TOKEN               = 0x80090308
	SEC_E_LOGON_DENIED                = 0x8009030C
	SEC_E_NO_AUTHENTICATING_AUTHORITY = 0x80090311
	SEC_E_NO_CREDENTIALS              = 0x8009030E
	SEC_E_TARGET_UNKNOWN              = 0x80090303
	SEC_E_UNSUPPORTED_FUNCTION        = 0x80090302
	SEC_E_WRONG_PRINCIPAL             = 0x80090322
	SEC_E_NOT_OWNER                   = 0x80090306
	SEC_E_SECPKG_NOT_FOUND            = 0x80090305
	SEC_E_UNKNOWN_CREDENTIALS         = 0x8009030D
	SEC_E_RENEGOTIATE                 = 590625
	SEC_E_COMPLETE_AND_CONTINUE       = 590612
	SEC_E_COMPLETE_NEEDED             = 590611
	SEC_E_INCOMPLETE_CREDENTIALS      = 590624
)

// Credential use
const (
	SECPKG_CRED_INBOUND  = 1
	SECPKG_CRED_OUTBOUND = 2
	// ...
)

// For InitializeSecurityContext fContextReq / context attributes, you’ll define what you need:
const (
	ISC_REQ_MUTUAL_AUTH     = 0x00000002
	ISC_REQ_DELEGATE        = 0x00000001
	ISC_REQ_CONFIDENTIALITY = 0x00000010
	// etc.
)

// The structure versions / buffer types
const (
	SECBUFFER_VERSION = 0
	SECBUFFER_EMPTY   = 0
	SECBUFFER_DATA    = 1
	SECBUFFER_TOKEN   = 2
)

// For QueryContextAttributes
const (
	SECPKG_ATTR_SIZES       = 0
	SECPKG_ATTR_SESSION_KEY = 9
)

// --- Structs that mirror C# / Windows native ---

// SECURITY_HANDLE => typically two void* (pointers) in Windows.
type SecurityHandle struct {
	LowPart  uintptr
	HighPart uintptr
}

// SECURITY_INTEGER => 64-bit total, typically: LowPart uint32, HighPart int32
type SecurityInteger struct {
	LowPart  uint32
	HighPart int32
}

// SecBuffer in Go
type SecBuffer struct {
	cbBuffer   uint32
	BufferType uint32
	pvBuffer   *byte
}

// SecBufferDesc in Go
type SecBufferDesc struct {
	ulVersion uint32
	cBuffers  uint32
	pBuffers  *SecBuffer
}

// SecPkgContext_Sizes
type SecPkgContextSizes struct {
	cbMaxToken        uint32
	cbMaxSignature    uint32
	cbBlockSize       uint32
	cbSecurityTrailer uint32
}

// SecPkgContext_SessionKey
type SecPkgContextSessionKey struct {
	SessionKeyLength uint32
	SessionKey       *byte
}

// --- Lazy loading secur32.dll & function pointers ---

var (
	secur32 = syscall.NewLazyDLL("secur32.dll")

	procAcquireCredentialsHandleW  = secur32.NewProc("AcquireCredentialsHandleW")
	procInitializeSecurityContextW = secur32.NewProc("InitializeSecurityContextW")
	procQueryContextAttributesW    = secur32.NewProc("QueryContextAttributesW")
	procEncryptMessage             = secur32.NewProc("EncryptMessage")
	procDecryptMessage             = secur32.NewProc("DecryptMessage")
	// etc. if needed
)

// AcquireCredentialsHandle (Unicode variant)
func AcquireCredentialsHandle(
	principal *uint16,
	secPackage *uint16,
	credentialUse uint32,
	logonID uintptr,
	authData uintptr,
	getKeyFn uintptr,
	getKeyArg uintptr,
	phCredential *SecurityHandle,
	ptsExpiry *SecurityInteger,
) syscall.Errno {
	r1, _, e1 := procAcquireCredentialsHandleW.Call(
		uintptr(unsafe.Pointer(principal)),
		uintptr(unsafe.Pointer(secPackage)),
		uintptr(credentialUse),
		uintptr(logonID),
		uintptr(authData),
		uintptr(getKeyFn),
		uintptr(getKeyArg),
		uintptr(unsafe.Pointer(phCredential)),
		uintptr(unsafe.Pointer(ptsExpiry)),
	)
	e1 = e1                  // avoid "declared and not used" error
	return syscall.Errno(r1) // The return code is in r1, e1 is usually the OS error code
}

// InitializeSecurityContext (Unicode variant, first call)
func InitializeSecurityContext_1(
	phCredential *SecurityHandle,
	phContext uintptr,
	pszTargetName *uint16,
	fContextReq uint32,
	reserved1 uint32,
	targetDataRep uint32,
	pInput uintptr,
	reserved2 uint32,
	phNewContext *SecurityHandle,
	pOutput *SecBufferDesc,
	pfContextAttr *uint32,
	ptsExpiry *SecurityInteger,
) syscall.Errno {
	r1, _, _ := procInitializeSecurityContextW.Call(
		uintptr(unsafe.Pointer(phCredential)),
		uintptr(phContext),
		uintptr(unsafe.Pointer(pszTargetName)),
		uintptr(fContextReq),
		uintptr(reserved1),
		uintptr(targetDataRep),
		uintptr(pInput),
		uintptr(reserved2),
		uintptr(unsafe.Pointer(phNewContext)),
		uintptr(unsafe.Pointer(pOutput)),
		uintptr(unsafe.Pointer(pfContextAttr)),
		uintptr(unsafe.Pointer(ptsExpiry)),
	)
	return syscall.Errno(r1)
}

// InitializeSecurityContext (subsequent calls)
func InitializeSecurityContext_2(
	phCredential *SecurityHandle,
	phContext *SecurityHandle,
	pszTargetName *uint16,
	fContextReq uint32,
	reserved1 uint32,
	targetDataRep uint32,
	pInput *SecBufferDesc,
	reserved2 uint32,
	phNewContext *SecurityHandle,
	pOutput *SecBufferDesc,
	pfContextAttr *uint32,
	ptsExpiry *SecurityInteger,
) syscall.Errno {
	r1, _, _ := procInitializeSecurityContextW.Call(
		uintptr(unsafe.Pointer(phCredential)),
		uintptr(unsafe.Pointer(phContext)),
		uintptr(unsafe.Pointer(pszTargetName)),
		uintptr(fContextReq),
		uintptr(reserved1),
		uintptr(targetDataRep),
		uintptr(unsafe.Pointer(pInput)),
		uintptr(reserved2),
		uintptr(unsafe.Pointer(phNewContext)),
		uintptr(unsafe.Pointer(pOutput)),
		uintptr(unsafe.Pointer(pfContextAttr)),
		uintptr(unsafe.Pointer(ptsExpiry)),
	)
	return syscall.Errno(r1)
}

// QueryContextAttributes (for SIZES, SESSION_KEY, etc.)
func QueryContextAttributes(
	phContext *SecurityHandle,
	ulAttribute uint32,
	pBuffer unsafe.Pointer,
) syscall.Errno {
	r1, _, _ := procQueryContextAttributesW.Call(
		uintptr(unsafe.Pointer(phContext)),
		uintptr(ulAttribute),
		uintptr(pBuffer),
	)
	return syscall.Errno(r1)
}

// EncryptMessage
func EncryptMessage(
	phContext *SecurityHandle,
	fQOP uint32,
	pMessage *SecBufferDesc,
	messageSeqNo uint32,
) syscall.Errno {
	r1, _, _ := procEncryptMessage.Call(
		uintptr(unsafe.Pointer(phContext)),
		uintptr(fQOP),
		uintptr(unsafe.Pointer(pMessage)),
		uintptr(messageSeqNo),
	)
	return syscall.Errno(r1)
}

// DecryptMessage
func DecryptMessage(
	phContext *SecurityHandle,
	pMessage *SecBufferDesc,
	messageSeqNo uint32,
	pfQOP *uint32,
) syscall.Errno {
	r1, _, _ := procDecryptMessage.Call(
		uintptr(unsafe.Pointer(phContext)),
		uintptr(unsafe.Pointer(pMessage)),
		uintptr(messageSeqNo),
		uintptr(unsafe.Pointer(pfQOP)),
	)
	return syscall.Errno(r1)
}

// getLastErrorString retrieves the last Windows error as a string.
func getLastErrorString() string {
	//err := windows.GetLastError()
	//if err == windows.ERROR_SUCCESS {
	//    return "No error occurred."
	//}
	//return fmt.Sprintf("Windows Error %d: %s", err, err.Error())
	return ""
}

// --- Helper functions for wide string conversions, etc. ---

func UTF16PtrFromString(s string) (*uint16, error) {
	return syscall.UTF16PtrFromString(s)
}

// --- Example: SSPISession struct that roughly mirrors the C# class ---

type SSPISession struct {
	phCredential   SecurityHandle
	hClientContext SecurityHandle
	ptsExpiry      SecurityInteger
	contextAttrs   uint32
	targetDataRep  uint32 // or int
	// track whether we've acquired credentials / security context
	hasCredHandle bool
	hasSecContext bool
}

// NewSSPISession is an example constructor
func NewSSPISession() *SSPISession {
	return &SSPISession{
		targetDataRep: 0, // default?
		hasCredHandle: false,
		hasSecContext: false,
	}
}

// AcquireCreds is analogous to AcquireCredentialsHandle for a user
func (s *SSPISession) AcquireCreds(username, packageName string, credUsage uint32) error {
	userPtr, _ := UTF16PtrFromString(username)
	pkgPtr, _ := UTF16PtrFromString(packageName)

	errno := AcquireCredentialsHandle(
		userPtr,
		pkgPtr,
		credUsage,
		0,
		0,
		0,
		0,
		&s.phCredential,
		&s.ptsExpiry,
	)
	if errno != 0 {
		return fmt.Errorf("AcquireCredentialsHandle failed (errno=%d): %v", errno, getLastErrorString())
	}
	s.hasCredHandle = true
	return nil
}

// InitializeSecurityContext (NTLM/Kerberos) first call
func (s *SSPISession) InitSecContextFirst(
	targetName string,
	contextReq uint32,
) (int, []byte, error) {
	if !s.hasCredHandle {
		return -1, nil, errors.New("credentials not acquired yet")
	}

	targetPtr, _ := UTF16PtrFromString(targetName)

	var outBuf SecBuffer
	outBufDesc := SecBufferDesc{
		ulVersion: SECBUFFER_VERSION,
		cBuffers:  1,
		pBuffers:  &outBuf,
	}
	// Allocate space for the token (like MAX_TOKEN_SIZE)
	bufSize := uint32(12288)
	mem := make([]byte, bufSize)
	// set up outBuf
	outBuf.cbBuffer = bufSize
	outBuf.BufferType = SECBUFFER_TOKEN
	// point pvBuffer -> &mem[0]
	outBuf.pvBuffer = &mem[0]

	errno := InitializeSecurityContext_1(
		&s.phCredential,
		0, // no existing context
		targetPtr,
		contextReq,
		0,
		s.targetDataRep,
		0,
		0,
		&s.hClientContext,
		&outBufDesc,
		&s.contextAttrs,
		&s.ptsExpiry,
	)
	if errno != 0 &&
		errno != SEC_E_CONTINUE_NEEDED &&
		errno != SEC_E_COMPLETE_AND_CONTINUE &&
		errno != SEC_E_COMPLETE_NEEDED &&
		errno != SEC_E_INCOMPLETE_CREDENTIALS {
		return int(errno), nil, fmt.Errorf("InitializeSecurityContext (first) failed (errno=%d): %v", errno, getLastErrorString())
	}

	// figure out how many bytes actually got written to outBuf
	var returnedBytes []byte
	if outBuf.cbBuffer > 0 && outBuf.cbBuffer <= bufSize {
		returnedBytes = mem[:outBuf.cbBuffer]
	}

	// Mark that we have a security context now
	s.hasSecContext = true

	return int(errno), returnedBytes, nil
}

// InitializeSecurityContext (NTLM/Kerberos) subsequent calls
func (s *SSPISession) InitSecContextNext(
	targetName string,
	contextReq uint32,
	inToken []byte,
) (int, []byte, error) {
	if !s.hasSecContext {
		return -1, nil, errors.New("security context not established yet")
	}

	targetPtr, _ := UTF16PtrFromString(targetName)

	// inBuf
	var inSecBuffer SecBuffer
	inSecBuffer.cbBuffer = uint32(len(inToken))
	inSecBuffer.BufferType = SECBUFFER_TOKEN
	var inMem []byte
	if len(inToken) > 0 {
		inMem = make([]byte, len(inToken))
		copy(inMem, inToken)
		inSecBuffer.pvBuffer = &inMem[0]
	}

	inBufDesc := SecBufferDesc{
		ulVersion: SECBUFFER_VERSION,
		cBuffers:  1,
		pBuffers:  &inSecBuffer,
	}

	// outBuf
	var outBuf SecBuffer
	outBufDesc := SecBufferDesc{
		ulVersion: SECBUFFER_VERSION,
		cBuffers:  1,
		pBuffers:  &outBuf,
	}
	outSize := uint32(12288)
	outMem := make([]byte, outSize)
	outBuf.cbBuffer = outSize
	outBuf.BufferType = SECBUFFER_TOKEN
	outBuf.pvBuffer = &outMem[0]

	errno := InitializeSecurityContext_2(
		&s.phCredential,
		&s.hClientContext,
		targetPtr,
		contextReq,
		0,
		s.targetDataRep,
		&inBufDesc,
		0,
		&s.hClientContext,
		&outBufDesc,
		&s.contextAttrs,
		&s.ptsExpiry,
	)
	if errno != 0 &&
		errno != SEC_E_CONTINUE_NEEDED &&
		errno != SEC_E_COMPLETE_AND_CONTINUE &&
		errno != SEC_E_COMPLETE_NEEDED &&
		errno != SEC_E_INCOMPLETE_CREDENTIALS {
		return int(errno), nil, fmt.Errorf("InitializeSecurityContext (subsequent) failed (errno=%d): %v", errno, getLastErrorString())
	}

	// how many bytes were written?
	var returnedBytes []byte
	if outBuf.cbBuffer > 0 && outBuf.cbBuffer <= outSize {
		returnedBytes = outMem[:outBuf.cbBuffer]
	}

	return int(errno), returnedBytes, nil
}

// Example: Query the session key
func (s *SSPISession) QuerySessionKey() (int, []byte, error) {
	if !s.hasSecContext {
		return -1, nil, errors.New("security context not established yet")
	}

	var sessionKey SecPkgContextSessionKey

	errno := QueryContextAttributes(
		&s.hClientContext,
		SECPKG_ATTR_SESSION_KEY,
		unsafe.Pointer(&sessionKey),
	)
	if errno != 0 {
		return -1, nil, fmt.Errorf("QueryContextAttributes(SESSION_KEY) failed (errno=%d): %v", errno, getLastErrorString())
	}

	// Copy out the key
	length := sessionKey.SessionKeyLength
	if length == 0 || sessionKey.SessionKey == nil {
		return -1, nil, errors.New("no session key returned")
	}

	// Make a Go slice
	data := make([]byte, length)
	copy(data, (*[1 << 30]byte)(unsafe.Pointer(sessionKey.SessionKey))[:length:length])

	return int(errno), data, nil
}

// Example: Using EncryptMessage to see sequence number, etc.
func (s *SSPISession) EncryptTestMessage() ([]byte, error) {
	if !s.hasSecContext {
		return nil, errors.New("security context not established yet")
	}

	// For demonstration, we pass in a small message
	plainMsg := []byte("Hello, SSPI!")

	// We first query sizes:
	var sizes SecPkgContextSizes
	errno := QueryContextAttributes(
		&s.hClientContext,
		SECPKG_ATTR_SIZES,
		unsafe.Pointer(&sizes),
	)
	if errno != 0 {
		return nil, fmt.Errorf("QueryContextAttributes(SIZES) failed (errno=%d): %v", errno, getLastErrorString())
	}

	// Prepare 2 buffers: [SECBUFFER_DATA, SECBUFFER_TOKEN]
	dataBuf := SecBuffer{
		cbBuffer:   uint32(len(plainMsg)),
		BufferType: SECBUFFER_DATA,
	}
	dataMem := make([]byte, len(plainMsg))
	copy(dataMem, plainMsg)
	dataBuf.pvBuffer = &dataMem[0]

	tokenBuf := SecBuffer{
		cbBuffer:   sizes.cbSecurityTrailer,
		BufferType: SECBUFFER_TOKEN,
	}
	tokenMem := make([]byte, sizes.cbSecurityTrailer)
	tokenBuf.pvBuffer = &tokenMem[0]

	// Build descriptor
	bufs := []SecBuffer{dataBuf, tokenBuf}
	outDesc := SecBufferDesc{
		ulVersion: SECBUFFER_VERSION,
		cBuffers:  2,
		pBuffers:  &bufs[0],
	}

	// Encrypt
	errno = EncryptMessage(&s.hClientContext, 0, &outDesc, 0)
	if errno != 0 {
		return nil, fmt.Errorf("EncryptMessage failed (errno=%d): %v", errno, getLastErrorString())
	}

	// The encrypted data is now in dataBuf + tokenBuf
	// Combine them if you want them in one slice
	encrypted := append(tokenMem, dataMem...)
	return encrypted, nil
}

func (s *SSPISession) processMessage(message WSPacket) (*WSPacket, error) {
	switch message.CmdType {
	case 11:
		// NTLM auth step 1
		authmsg, err := WSNNTLMAuthFromBytes(message.Data.([]byte))
		if err != nil {
			return nil, fmt.Errorf("failed to parse NTLM auth message: %w", err)
		}

		// Process the message
		// ...
		if err := s.AcquireCreds(authmsg.Username, "NTLM", uint32(authmsg.CredUsage)); err != nil {
			return nil, fmt.Errorf("failed to acquire credentials: %w", err)
		}
		errno, outToken, err := s.InitSecContextFirst(authmsg.TargetName, uint32(authmsg.CtxAttr))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize security context: %w", err)
		}

		reply := WSNNTLMAuthReply{
			Status:   errno,
			CtxAttr:  int(s.contextAttrs),
			AuthData: outToken,
		}

		data, err := reply.ToData()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize NTLM auth reply: %w", err)
		}

		dataPacket := WSPacket{
			Length:  (uint32)(22 + len(data)),
			CmdType: 12,
			Token:   message.Token,
			Data:    data,
		}
		return &dataPacket, nil

	case 13:
		// NTLM auth step 2

		challenge, err := WSNNTLMChallengeFromBytes(message.Data.([]byte))
		if err != nil {
			return nil, fmt.Errorf("failed to parse NTLM challenge: %w", err)
		}

		// Process the message
		// ...
		errno, outToken, err := s.InitSecContextNext(challenge.TargetName, uint32(challenge.CtxAttr), challenge.AuthData)
		if err != nil {
			return nil, fmt.Errorf("failed to continue security context: %w", err)
		}

		reply := WSNNTLMChallengeReply{
			Status:   errno,
			CtxAttr:  int(s.contextAttrs),
			AuthData: outToken,
		}

		data, err := reply.ToData()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize NTLM challenge reply: %w", err)
		}

		dataPacket := WSPacket{
			Length:  (uint32)(22 + len(data)),
			CmdType: 14,
			Token:   message.Token,
			Data:    data,
		}

		return &dataPacket, nil

	case 15:
		{
			// Kerberos auth step 1
			authmsg, err := WSNKerberosAuthFromBytes(message.Data.([]byte))
			if err != nil {
				return nil, fmt.Errorf("failed to parse Kerberos auth message: %w", err)
			}

			fmt.Println(authmsg)

			// Process the message
			// ...
			if err := s.AcquireCreds(authmsg.Username, "Kerberos", uint32(authmsg.CredUsage)); err != nil {
				return nil, fmt.Errorf("failed to acquire credentials: %w", err)
			}

			errno, outToken, err := s.InitSecContextFirst(authmsg.TargetName, uint32(authmsg.CtxAttr))
			if err != nil {
				return nil, fmt.Errorf("failed to initialize security context: %w", err)
			}

			reply := WSNKerberosAuthReply{
				Status:   errno,
				CtxAttr:  int(s.contextAttrs),
				AuthData: outToken,
			}

			data, err := reply.ToData()
			if err != nil {
				return nil, fmt.Errorf("failed to serialize Kerberos auth reply: %w", err)
			}

			dataPacket := WSPacket{
				Length:  (uint32)(22 + len(data)),
				CmdType: 16,
				Token:   message.Token,
				Data:    data,
			}
			return &dataPacket, nil
		}
	case 17:
		{
			// session key
			errno, key, err := s.QuerySessionKey()
			if err != nil {
				return nil, fmt.Errorf("failed to query session key: %w", err)
			}

			reply := WSNSessionKeyReply{
				Status:     errno,
				SessionKey: key,
			}

			data, err := reply.ToData()
			if err != nil {
				return nil, fmt.Errorf("failed to serialize session key reply: %w", err)
			}

			dataPacket := WSPacket{
				Length:  (uint32)(22 + len(data)),
				CmdType: 18,
				Token:   message.Token,
				Data:    data,
			}

			return &dataPacket, nil
		}
	case 19:
		{
			// Encrypt test message
			encrypted, err := s.EncryptTestMessage()
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt test message: %w", err)
			}

			reply := WSNGetSequenceNoReply{
				Status:  0,
				EncData: encrypted,
			}

			data, err := reply.ToData()
			if err != nil {
				return nil, fmt.Errorf("failed to serialize encrypted message reply: %w", err)
			}

			dataPacket := WSPacket{
				Length:  (uint32)(22 + len(data)),
				CmdType: 20,
				Token:   message.Token,
				Data:    data,
			}

			return &dataPacket, nil
		}
	}

	return nil, errors.New("unknown command type")
}

func localSSPITest() {
	session := NewSSPISession()

	// Acquire credentials for "NTLM" or "Kerberos" package, for example
	if err := session.AcquireCreds("MyUsername", "NTLM", SECPKG_CRED_OUTBOUND); err != nil {
		fmt.Println("AcquireCreds error:", err)
		return
	}

	// Initialize security context (first call)
	errno, outToken, err := session.InitSecContextFirst("host/machine", ISC_REQ_CONFIDENTIALITY)
	if err != nil {
		fmt.Println("InitSecContextFirst error:", err)
		return
	}
	fmt.Println("First token:", outToken)
	errno = errno // avoid "declared and not used" error

	// Suppose we got a challenge from the server. We call next:
	challenge := []byte{ /* server's challenge token bytes */ }
	errno, outToken2, err := session.InitSecContextNext("host/machine", ISC_REQ_CONFIDENTIALITY, challenge)
	if err != nil {
		fmt.Println("InitSecContextNext error:", err)
		return
	}
	fmt.Println("Next token:", outToken2)

	// Suppose at this point we are authenticated. Query the session key:
	errno, key, err := session.QuerySessionKey()
	if err != nil {
		fmt.Println("QuerySessionKey error:", err)
		return
	}
	errno = errno // avoid "declared and not used" error
	fmt.Printf("Session key: %x\n", key)

	// Test encryption:
	encrypted, err := session.EncryptTestMessage()
	if err != nil {
		fmt.Println("EncryptTestMessage error:", err)
		return
	}
	fmt.Printf("Encrypted data: %x\n", encrypted)
}
