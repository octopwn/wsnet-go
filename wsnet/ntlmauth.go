package wsnet

import (
    "bytes"
    "fmt"
    "strconv"
)


type WSNNTLMAuth struct {
    Username    string
    CredUsage   int
    CtxAttr     int
    TargetName  string
}

type WSNNTLMAuthReply struct {
    Status    int
    CtxAttr   int
    AuthData  []byte
}


func WSNNTLMAuthFromBytes(b []byte) (*WSNNTLMAuth, error) {
    buf := bytes.NewReader(b)
    var auth WSNNTLMAuth

    // 1. Read Username
    username, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read username: %w", err)
    }
    auth.Username = username

    // 2. Read CredUsage as string and convert to int
    credUsageStr, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read credusage: %w", err)
    }
    credUsage, err := strconv.Atoi(credUsageStr)
    if err != nil {
        return nil, fmt.Errorf("failed to convert credusage to int: %w", err)
    }
    auth.CredUsage = credUsage

    // 3. Read CtxAttr as string and convert to int
    ctxAttrStr, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read ctxattr: %w", err)
    }
    ctxAttr, err := strconv.Atoi(ctxAttrStr)
    if err != nil {
        return nil, fmt.Errorf("failed to convert ctxattr to int: %w", err)
    }
    auth.CtxAttr = ctxAttr

    // 4. Read TargetName
    targetName, err := readStr(buf)
    if err != nil {
        return nil, fmt.Errorf("failed to read targetname: %w", err)
    }
    auth.TargetName = targetName

    return &auth, nil
}


func (w *WSNNTLMAuth) ToData() ([]byte, error) {
    var buf bytes.Buffer

    // 1. Write Username
    if err := writeStr(&buf, w.Username); err != nil {
        return nil, fmt.Errorf("failed to write username: %w", err)
    }

    // 2. Write CredUsage as string
    credUsageStr := strconv.Itoa(w.CredUsage)
    if err := writeStr(&buf, credUsageStr); err != nil {
        return nil, fmt.Errorf("failed to write credusage: %w", err)
    }

    // 3. Write CtxAttr as string
    ctxAttrStr := strconv.Itoa(w.CtxAttr)
    if err := writeStr(&buf, ctxAttrStr); err != nil {
        return nil, fmt.Errorf("failed to write ctxattr: %w", err)
    }

    // 4. Write TargetName
    if err := writeStr(&buf, w.TargetName); err != nil {
        return nil, fmt.Errorf("failed to write targetname: %w", err)
    }

    return buf.Bytes(), nil
}



func WSNNTLMAuthReplyFromBytes(b []byte) (*WSNNTLMAuthReply, error) {
    buf := bytes.NewReader(b)
    var reply WSNNTLMAuthReply

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



func (w *WSNNTLMAuthReply) ToData() ([]byte, error) {
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
