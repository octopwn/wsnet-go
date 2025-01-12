// basicinfo_win.go
//go:build windows
// +build windows

package wsnet

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Constants
const (
	TOKEN_QUERY                = 0x0008
	TokenStatistics            = 10
	TokenUser                  = 1
	LsaGetLogonSessionDataProc = "LsaGetLogonSessionData"
	LsaFreeReturnBufferProc    = "LsaFreeReturnBuffer"
)

// Structures

type TOKEN_STATISTICS struct {
	TokenId            windows.LUID
	AuthenticationId   windows.LUID
	ExpirationTime     int64
	TokenType          uint32
	ImpersonationLevel uint32
	EffectiveOnly      uint32
	SessionId          uint32
	OriginatingLogonId windows.LUID
	PackageId          windows.LUID
	ModifiedId         windows.LUID
}

type UNICODE_STRING struct {
	Length        uint16
	MaximumLength uint16
	Buffer        *uint16
}

type SECURITY_LOGON_SESSION_DATA struct {
	Size                       uint32
	LoginId                    windows.LUID
	LogonType                  uint32
	AuthenticationPackage      uint32
	AuthenticationPackageName  UNICODE_STRING
	TransmittedServices        UNICODE_STRING
	Source                     UNICODE_STRING
	Destination                UNICODE_STRING
	PacketQualityOfService     uint32
	LogonTime                  int64
	LogonServer                UNICODE_STRING
	AuthenticationId           windows.LUID
	UserName                   UNICODE_STRING
	WorkstationName            UNICODE_STRING
	DnsDomainName              UNICODE_STRING
	Upn                        UNICODE_STRING
	LogonDomainId              windows.LUID
	PrimaryDomainId            windows.LUID
	LogonUserSid               uintptr
	LogonUserName              UNICODE_STRING
	LogonDomainName            UNICODE_STRING
	AuthenticationId2          windows.LUID
	ProcessedLogonId           windows.LUID
	LogonToken                 uintptr
	AuthenticationPackageName2 UNICODE_STRING
	TransmittedServices2       UNICODE_STRING
	Source2                    UNICODE_STRING
	ProfilePath                UNICODE_STRING
	HomeDirectory              UNICODE_STRING
	HomeDirectoryDrive         UNICODE_STRING
	LogonScript                UNICODE_STRING
	Parameters                 UNICODE_STRING
	WorkstationName2           UNICODE_STRING
	LogonTime2                 int64
	Sid                        uintptr
}

type LUID struct {
	LowPart  uint32
	HighPart int32
}

type LogonInfo struct {
	Domain    string
	FQDN      string
	IPAddress string
	Hostname  string
	UserSID   string
	Username  string
}

func getLogonInfo() (*LogonInfo, error) {
	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("os.Hostname failed: %v", err)
	}

	// Open process token
	var token windows.Token
	err = windows.OpenProcessToken(windows.CurrentProcess(), TOKEN_QUERY, &token)
	if err != nil {
		return nil, fmt.Errorf("OpenProcessToken failed: %v", err)
	}
	defer token.Close()

	// Get TokenStatistics
	var tokenStatsSize uint32
	// First call to get the size
	err = windows.GetTokenInformation(token, TokenStatistics, nil, 0, &tokenStatsSize)
	if err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return nil, fmt.Errorf("GetTokenInformation (size) failed: %v", err)
	}

	tokenStats := make([]byte, tokenStatsSize)
	err = windows.GetTokenInformation(token, TokenStatistics, &tokenStats[0], tokenStatsSize, &tokenStatsSize)
	if err != nil {
		return nil, fmt.Errorf("GetTokenInformation failed: %v", err)
	}

	// Marshal TOKEN_STATISTICS
	tokenStatsStruct := (*TOKEN_STATISTICS)(unsafe.Pointer(&tokenStats[0]))

	// Load secur32.dll for LsaGetLogonSessionData and LsaFreeReturnBuffer
	secur32 := windows.NewLazySystemDLL("secur32.dll")
	lsaGetLogonSessionDataProc := secur32.NewProc(LsaGetLogonSessionDataProc)
	lsaFreeReturnBufferProc := secur32.NewProc(LsaFreeReturnBufferProc)

	if err := secur32.Load(); err != nil {
		return nil, fmt.Errorf("Failed to load secur32.dll: %v", err)
	}

	// Prepare LogonSessionId (which is a LUID)
	logonSessionId := tokenStatsStruct.AuthenticationId

	// Call LsaGetLogonSessionData
	var pSessionData uintptr
	ret, _, err := lsaGetLogonSessionDataProc.Call(uintptr(unsafe.Pointer(&logonSessionId)), uintptr(unsafe.Pointer(&pSessionData)))
	if ret != 0 {
		return nil, fmt.Errorf("LsaGetLogonSessionData failed: %v", err)
	}
	defer lsaFreeReturnBufferProc.Call(pSessionData)

	// Marshal SECURITY_LOGON_SESSION_DATA
	sessionData := (*SECURITY_LOGON_SESSION_DATA)(unsafe.Pointer(pSessionData))

	// Extract Domain
	var domain string
	if sessionData.DnsDomainName.Length > 0 && sessionData.DnsDomainName.Buffer != nil {
		domain = windows.UTF16PtrToString(sessionData.DnsDomainName.Buffer)
	}

	// Extract LogonServer
	var logonserver, fqdn, ipAddress string
	if sessionData.LogonServer.Length > 0 && sessionData.LogonServer.Buffer != nil {
		logonserver = windows.UTF16PtrToString(sessionData.LogonServer.Buffer)
		logonserver = strings.TrimSpace(logonserver)
		if logonserver != "" {
			// Perform DNS lookup
			hostEntries, err := net.LookupHost(logonserver)
			if err == nil && len(hostEntries) > 0 {
				// LookupHost returns IP addresses; to get FQDN, use LookupAddr
				ipAddress = hostEntries[0]
				names, err := net.LookupAddr(hostEntries[0])
				if err == nil && len(names) > 0 {
					fqdn = strings.TrimSuffix(names[0], ".")
				}
			}
		}
	}

	// Get User SID and Username
	var userSID string
	var username string

	// Retrieve TokenUser information
	var tokenUserSize uint32
	err = windows.GetTokenInformation(token, TokenUser, nil, 0, &tokenUserSize)
	if err != nil && err != windows.ERROR_INSUFFICIENT_BUFFER {
		return nil, fmt.Errorf("GetTokenInformation (TokenUser size) failed: %v", err)
	}

	tokenUserData := make([]byte, tokenUserSize)
	err = windows.GetTokenInformation(token, TokenUser, &tokenUserData[0], tokenUserSize, &tokenUserSize)
	if err != nil {
		return nil, fmt.Errorf("GetTokenInformation (TokenUser) failed: %v", err)
	}

	// Marshal TOKEN_USER structure
	type SIDAndAttributes struct {
		Sid        *windows.SID
		Attributes uint32
	}

	tokenUser := (*SIDAndAttributes)(unsafe.Pointer(&tokenUserData[0]))
	// Convert SID to string
	userSID = tokenUser.Sid.String()

	// Get Username
	// Use LookupAccountSid to get the username associated with the SID
	var nameLen uint32 = 0
	var domainLen uint32 = 0
	var peUse uint32

	// First call to determine buffer sizes
	windows.LookupAccountSid(nil, tokenUser.Sid, nil, &nameLen, nil, &domainLen, &peUse)
	// Allocate buffers
	name := make([]uint16, nameLen)
	domainName := make([]uint16, domainLen)
	// Second call to get actual data
	err = windows.LookupAccountSid(nil, tokenUser.Sid, &name[0], &nameLen, &domainName[0], &domainLen, &peUse)
	if err != nil {
		return nil, fmt.Errorf("LookupAccountSid failed: %v", err)
	}
	username = fmt.Sprintf("%s\\%s", windows.UTF16ToString(domainName), windows.UTF16ToString(name))
	if domain == "" {
		domain = windows.UTF16ToString(domainName)
	}

	// Assemble the result
	result := &LogonInfo{
		Domain:    domain,
		FQDN:      fqdn,
		IPAddress: ipAddress,
		Hostname:  hostname,
		UserSID:   userSID,
		Username:  username,
	}
	return result, nil
}

func BuildGetInfoReply() (*WSNGetInfoReply, error) {
	logon, err := getLogonInfo()
	if err != nil {
		return nil, fmt.Errorf("Failed to get logon info: %v", err)
	}

	info := &WSNGetInfoReply{
		pid:           strconv.Itoa(os.Getpid()),
		username:      logon.Username,
		domain:        logon.Domain,
		logonserver:   logon.FQDN,
		cpuarch:       "X64",
		hostname:      logon.Hostname,
		usersid:       logon.UserSID,
		os:            "WINDOWS",
		logonserverip: logon.IPAddress,
	}
	return info, nil
}
