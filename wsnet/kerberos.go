package wsnet

import (
    "bytes"
    "fmt"
    "strconv"
)


type WSNKerberosAuth struct {
    Username    string
    CredUsage   int
    CtxAttr     int
    TargetName  string
    AuthData    []byte
}


func WSNKerberosAuthFromBytes(b []byte) (*WSNKerberosAuth, error) {
    buf := bytes.NewReader(b)
    var auth WSNKerberosAuth

    // 1. Read Username
    username, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read Username: %w", err)
    }
    auth.Username = username

    // 2. Read CredUsage as string and convert to int
    credUsageStr, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read CredUsage: %w", err)
    }
    credUsage, err := strconv.Atoi(credUsageStr)
    if err != nil {
        return nil, fmt.Errorf("failed to convert CredUsage to int: %w", err)
    }
    auth.CredUsage = credUsage

    // 3. Read CtxAttr as string and convert to int
    ctxAttrStr, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read CtxAttr: %w", err)
    }
    ctxAttr, err := strconv.Atoi(ctxAttrStr)
    if err != nil {
        return nil, fmt.Errorf("failed to convert CtxAttr to int: %w", err)
    }
    auth.CtxAttr = ctxAttr

    // 4. Read TargetName
    targetName, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read TargetName: %w", err)
    }
    auth.TargetName = targetName

    // 5. Read AuthData
    authData, err := readBytes(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read AuthData: %w", err)
    }
    auth.AuthData = authData

    return &auth, nil
}


func (w *WSNKerberosAuth) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // 1. Write Username
    if err := writeStr(&buf, w.Username); err != nil {
        return nil, fmt.Errorf("failed to write Username: %w", err)
    }

    // 2. Write CredUsage as string
    credUsageStr := strconv.Itoa(w.CredUsage)
    if err := writeStr(&buf, credUsageStr); err != nil {
        return nil, fmt.Errorf("failed to write CredUsage: %w", err)
    }

    // 3. Write CtxAttr as string
    ctxAttrStr := strconv.Itoa(w.CtxAttr)
    if err := writeStr(&buf, ctxAttrStr); err != nil {
        return nil, fmt.Errorf("failed to write CtxAttr: %w", err)
    }

    // 4. Write TargetName
    if err := writeStr(&buf, w.TargetName); err != nil {
        return nil, fmt.Errorf("failed to write TargetName: %w", err)
    }

    // 5. Write AuthData
    if err := writeBytes(&buf, w.AuthData); err != nil {
        return nil, fmt.Errorf("failed to write AuthData: %w", err)
    }

    return buf.Bytes(), nil
}


type WSNKerberosAuthReply struct {
    Status    int
    CtxAttr   int
    AuthData  []byte
}


func WSNKerberosAuthReplyFromBytes(b []byte) (*WSNKerberosAuthReply, error) {
    buf := bytes.NewReader(b)
    var reply WSNKerberosAuthReply

    // 1. Read Status
    status, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read Status: %w", err)
    }
    reply.Status, err = strconv.Atoi(status)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Status to int: %w", err)
	}
	
    // 2. Read CtxAttr as string and convert to int
    ctxAttrStr, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read CtxAttr: %w", err)
    }
    ctxAttr, err := strconv.Atoi(ctxAttrStr)
    if err != nil {
        return nil, fmt.Errorf("failed to convert CtxAttr to int: %w", err)
    }
    reply.CtxAttr = ctxAttr

    // 3. Read AuthData
    authData, err := readBytes(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read AuthData: %w", err)
    }
    reply.AuthData = authData

    return &reply, nil
}

func (w *WSNKerberosAuthReply) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // 1. Write Status
	statusStr := strconv.Itoa(w.Status)
    if err := writeStr(&buf, statusStr); err != nil {
        return nil, fmt.Errorf("failed to write Status: %w", err)
    }

    // 2. Write CtxAttr as string
    ctxAttrStr := strconv.Itoa(w.CtxAttr)
    if err := writeStr(&buf, ctxAttrStr); err != nil {
        return nil, fmt.Errorf("failed to write CtxAttr: %w", err)
    }

    // 3. Write AuthData
    if err := writeBytes(&buf, w.AuthData); err != nil {
        return nil, fmt.Errorf("failed to write AuthData: %w", err)
    }

    return buf.Bytes(), nil
}