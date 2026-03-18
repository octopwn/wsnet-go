package wsnet

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"encoding/binary"
)

type ProcessHandle struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
}

type ClientHandler struct {
	send              func(WSPacket)
	connections       map[string]net.Conn
	fileHandles       map[string]*os.File
	serverSockets     map[string]net.Listener
	serverTokenToConn map[string]string
	connectionsMu     sync.Mutex // Mutex for synchronizing access to connections
	fileHandlesMu     sync.Mutex
	serverSocketsMu   sync.Mutex
	authHandlers      map[string]*SSPISession
	authHandlersMu    sync.Mutex
	processes         map[string]*ProcessHandle // Map token to process handle
	processesMu       sync.Mutex               // Mutex for process map
	outgoing          chan WSPacket            // Channel for outgoing messages
	inforeply         *WSNGetInfoReply
}

// Initialize the ClientHandler struct
func NewClientHandler() *ClientHandler {
	return &ClientHandler{
		connections:       make(map[string]net.Conn),
		connectionsMu:     sync.Mutex{},
		serverSockets:     make(map[string]net.Listener),
		serverSocketsMu:   sync.Mutex{},
		serverTokenToConn: make(map[string]string),
		fileHandles:       make(map[string]*os.File),
		fileHandlesMu:     sync.Mutex{},
		authHandlers:      make(map[string]*SSPISession),
		authHandlersMu:    sync.Mutex{},
		processes:         make(map[string]*ProcessHandle),
		processesMu:       sync.Mutex{},
		outgoing:          make(chan WSPacket, 1024), // Initialize the outgoing channel
	}
}

// Writer goroutine to handle outgoing messages
// This method is responsible to keep writing in a thread-safe way
// only this function can touch the outgoing channel on c.send
// do not call this function, only start it as a goroutine
func (c *ClientHandler) writePump() {
	for message := range c.outgoing {
		// Write the message to the WebSocket connection
		c.send(message)
	}
}

func (c *ClientHandler) OnConnect(sendfnt func(WSPacket)) {
	logVerbose("Client connected")
	c.send = sendfnt
	go c.writePump() // Start the writer goroutine
}

func (c *ClientHandler) OnDisconnect() {
	logVerbose("Client disconnected")
	close(c.outgoing) // Close the channel to stop the writer goroutine
	for token, conn := range c.connections {
		conn.Close()
		delete(c.connections, token)
	}
}

func (ch *ClientHandler) AddConnection(key string, conn net.Conn) {
	ch.connectionsMu.Lock()
	defer ch.connectionsMu.Unlock()
	ch.connections[key] = conn
}

func (ch *ClientHandler) AddFileHandle(key string, file *os.File) {
	ch.fileHandlesMu.Lock()
	defer ch.fileHandlesMu.Unlock()
	ch.fileHandles[key] = file
}

func (ch *ClientHandler) AddServerSocket(key string, conn net.Listener) {
	ch.serverSocketsMu.Lock()
	defer ch.serverSocketsMu.Unlock()
	ch.serverSockets[key] = conn
}

func (ch *ClientHandler) RemoveConnection(key string) {
	ch.connectionsMu.Lock()
	defer ch.connectionsMu.Unlock()
	delete(ch.connections, key)
}

func (ch *ClientHandler) RemoveServerSocket(key string) {
	ch.serverSocketsMu.Lock()
	defer ch.serverSocketsMu.Unlock()
	for k, v := range ch.serverTokenToConn {
		if v == key {
			ch.RemoveConnection(k)
		}
	}
	delete(ch.serverSockets, key)
}

func (ch *ClientHandler) RemoveFileHandle(key string) {
	ch.fileHandlesMu.Lock()
	defer ch.fileHandlesMu.Unlock()
	delete(ch.fileHandles, key)
}

func (ch *ClientHandler) RemoveAuthHandler(key string) {
	ch.authHandlersMu.Lock()
	defer ch.authHandlersMu.Unlock()
	delete(ch.authHandlers, key)
}

func (ch *ClientHandler) AddAuthHandler(key string, handler *SSPISession) {
	ch.authHandlersMu.Lock()
	defer ch.authHandlersMu.Unlock()
	ch.authHandlers[key] = handler
}

func (ch *ClientHandler) AddProcess(key string, cmd *exec.Cmd, stdin io.WriteCloser) {
	ch.processesMu.Lock()
	defer ch.processesMu.Unlock()
	ch.processes[key] = &ProcessHandle{
		cmd:   cmd,
		stdin: stdin,
	}
}

func (ch *ClientHandler) RemoveProcess(key string) {
	ch.processesMu.Lock()
	defer ch.processesMu.Unlock()
	if handle, ok := ch.processes[key]; ok {
		if handle.stdin != nil {
			handle.stdin.Close()
		}
		delete(ch.processes, key)
	}
}

func (ch *ClientHandler) GetProcess(key string) (*ProcessHandle, bool) {
	ch.processesMu.Lock()
	defer ch.processesMu.Unlock()
	handle, ok := ch.processes[key]
	return handle, ok
}

func (c *ClientHandler) OnMessage(message WSPacket) {
	logVerbosef("Received: %+v", message)
	switch message.CmdType {
	case 0:
		{
			key := fmt.Sprintf("%x", message.Token)
			if _, ok := c.connections[key]; ok {
				c.RemoveConnection(key)
				return
			}
			if _, ok := c.fileHandles[key]; ok {
				c.RemoveFileHandle(key)
				return
			}

			// if not in connections map, it could be in the future to be in the auth or file map but it's not implemented yet
		}
	case 1:
		{
			key := fmt.Sprintf("%x", message.Token)
			if conn, ok := c.connections[key]; ok {
				conn.Close()
				c.RemoveConnection(fmt.Sprintf("%x", message.Token))
				return
			}
			if _, ok := c.fileHandles[key]; ok {
				c.RemoveFileHandle(key)
				return
			}

			//same as OK
		}
	case 7:
		{
			key := fmt.Sprintf("%x", message.Token)
			if conn, ok := c.connections[key]; ok {
				conn.Write(message.Data.([]byte))
			}
		}
	case 8:
		{
			c.sendGetInfo(message.Token)
		}

	case 5:
		{
			if connData, ok := message.Data.(*WSNConnect); ok {
				if connData.Bind {
					if connData.Protocol == "UDP" {
						conn, err := net.ListenUDP("udp", nil)
						if err != nil {
							c.sendError(message.Token, err)
							return
						}
						tokenStr := fmt.Sprintf("%x", message.Token)
						c.AddConnection(tokenStr, conn)
						connectionToken := [16]byte{} // static for now
						go c.handleIncomingDataUDP(message.Token, tokenStr, connectionToken, conn, c)
						c.sendContinue(message.Token)
						return
					} else {
						conn, err := net.Listen("tcp", connData.IP+":"+strconv.Itoa(int(connData.Port)))
						if err != nil {
							c.sendError(message.Token, err)
							return
						}
						tokenStr := fmt.Sprintf("%x", message.Token)
						go c.handleTCPServer(message.Token, tokenStr, conn, c)
						c.AddServerSocket(tokenStr, conn)
						c.sendContinue(message.Token)
						return
					}
				}

				logVerbosef("Connecting to %s server at IP: %s Port: %d", connData.Protocol, connData.IP, connData.Port)
				if connData.Protocol == "TCP" {
					timeout := time.Duration(1000 * time.Millisecond)
					conn, err := net.DialTimeout("tcp", connData.IP+":"+strconv.Itoa(int(connData.Port)), timeout)
					if err != nil {
						c.sendError(message.Token, err)
						return
					}

					tokenStr := fmt.Sprintf("%x", message.Token)
					c.AddConnection(tokenStr, conn)

					// Start a goroutine to handle incoming data
					go c.handleIncomingData(message.Token, tokenStr, conn, c, nil)
					c.sendContinue(message.Token)

				} else {
					logVerbose("UDP Protocol")
					conn, err := net.ListenUDP("udp", nil)
					if err != nil {
						c.sendError(message.Token, err)
						return
					}

					tokenStr := fmt.Sprintf("%x", message.Token)
					// AddConnection expects net.Conn, so we can cast:
					c.AddConnection(tokenStr, conn)

					// Start a goroutine to handle incoming data from the UDP connection
					connectionToken := [16]byte{} // static for now
					go c.handleIncomingDataUDP(message.Token, tokenStr, connectionToken, conn, c)

					// Let the client know we’re connected
					c.sendContinue(message.Token)
				}
			} else {
				log.Println("Error parsing WSNConnect")
				c.sendError(message.Token, fmt.Errorf("Error parsing WSNConnect"))
			}
		}
	case 11, 15:
		{
			sspiSession := NewSSPISession()

			result, err := sspiSession.processMessage(message)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.AddAuthHandler(fmt.Sprintf("%x", message.Token), sspiSession)
			c.outgoing <- *result
			return
		}
	case 13, 17, 19:
		{
			logVerbose("SSPIAuth message received")

			// Get the sspi session
			sspiSession, ok := c.authHandlers[fmt.Sprintf("%x", message.Token)]
			if !ok {
				c.sendError(message.Token, fmt.Errorf("SSPI session not found"))
				return
			}
			result, err := sspiSession.processMessage(message)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.outgoing <- *result
			return
		}
	case 200:
		{
			// data to be dispached can be either an UDP client or a SERVER SOCKET
			// servers are not implemented yet

			logVerbose("Server Socket Data message received")
			connData, err := WSNClientSocketDataFromBytes(message.Data.([]byte))
			if err != nil {
				log.Println("Error parsing WSNServerSocketData")
				c.sendError(message.Token, fmt.Errorf("Error parsing WSNServerSocketData"))
				return
			}

			key := fmt.Sprintf("%x", message.Token)
			conn, ok := c.connections[key].(*net.UDPConn)
			if !ok {
				log.Println("Error getting connection")
				c.sendError(message.Token, fmt.Errorf("Error getting connection"))
				return
			}

			logVerbose("Sending data to client")
			dst, err := net.ResolveUDPAddr("udp", connData.ClientIP+":"+strconv.Itoa(int(connData.ClientPort)))
			if err != nil {
				log.Fatal(err)
			}

			_, err = conn.WriteTo(connData.Data, dst)
			if err != nil {
				log.Println("Error sending data to client")
				c.sendError(message.Token, fmt.Errorf("Error sending data to client"))
				return
			}
		}

	case 202:
		{
			go c.handleNameResolution(message.Token, message.Data.([]byte))
		}
	case 300:
		{
			req, err := WSNDirLSFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			go c.listDirectory(message.Token, req.Path)
		}
	case 301:
		{
			req, err := WSNDirMKFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			err = os.Mkdir(req.Path, 0755)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 302:
		{
			req, err := WSNDirRMFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			err = os.RemoveAll(req.Path)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)

		}
	case 303:
		{
			req, err := WSNDirCopyFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			err = CopyDirectory(req.SrcPath, req.DstPath)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)

		}
	case 304:
		{
			req, err := WSNDirMoveFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			err = os.Rename(req.SrcPath, req.DstPath)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 305:
		{
			req, err := WSNFileOpenFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			file, err := openFilePythonMode(req.Path, req.Mode)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}

			// Add the file handle to the map
			c.AddFileHandle(fmt.Sprintf("%x", message.Token), file)
			c.sendContinue(message.Token)

		}
	case 306:
		{
			logVerbose("File Read message received")
			req, err := WSNFileReadFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			logVerbose("Request:", req)
			// Read the file
			file, ok := c.fileHandles[fmt.Sprintf("%x", message.Token)]
			if !ok {
				c.sendError(message.Token, fmt.Errorf("File handle not found"))
				return
			}

			w, err := WSNFileDataFromFileData(req.Offset, req.Size, file)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			serializedData, err := w.ToData()
			if err != nil {
				c.sendError(message.Token, err)
				return
			}

			dataPacket := WSPacket{
				Length:  (uint32)(22 + len(serializedData)),
				CmdType: 307,
				Token:   message.Token,
				Data:    serializedData,
			}
			c.outgoing <- dataPacket
		}
	case 307:
		{
			req, err := WSNFileDataFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			file, ok := c.fileHandles[fmt.Sprintf("%x", message.Token)]
			if !ok {
				c.sendError(message.Token, fmt.Errorf("File handle not found"))
				return
			}

			_, err = file.WriteAt(req.Data, int64(req.Offset))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 309:
		{
			req, err := WSNFileCopyFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			err = copyFile(req.SrcPath, req.DstPath, nil)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)

		}
	case 310:
		{
			req, err := WSNFileMoveFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			err = os.Rename(req.SrcPath, req.DstPath)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 311:
		{
			req, err := WSNFileRMFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			err = os.Remove(req.Path)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 312:
		{
			logVerbose("File Stat message received")
			// nothing to parse, fetch file handle
			file, ok := c.fileHandles[fmt.Sprintf("%x", message.Token)]
			if !ok {
				c.sendError(message.Token, fmt.Errorf("File handle not found"))
				return
			}
			// Get file info
			info, err := file.Stat()
			if err != nil {
				c.sendError(message.Token, err)
				return
			}

			// Create a new fileentry struct
			entry := &WSNFileEntry{
				Root:  filepath.Dir(file.Name()),
				Name:  filepath.Base(file.Name()),
				IsDir: info.IsDir(),
				Size:  uint64(info.Size()),
				ATime: uint64(info.ModTime().Unix()),
				MTime: uint64(info.ModTime().Unix()),
				CTime: uint64(info.ModTime().Unix()),
			}
			// Serialize the file entry
			data, err := entry.ToData()
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			// Send the file entry to the client
			dataPacket := WSPacket{
				Length:  (uint32)(22 + len(data)),
				CmdType: 308,
				Token:   message.Token,
				Data:    data,
			}
			c.outgoing <- dataPacket
		}
	case 400: // Process list request
		{
			// No payload to parse
			go c.listProcesses(message.Token)
		}
	case 401: // Process start request
		{
			req, err := WSNProcessStartFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}

			logVerbosef("Received command: '%s', arguments: '%s'", req.Command, req.Arguments)

			// Parse command line - handle case where full command is in req.Command
			var command string
			var args []string
			
			// If Arguments field is empty or "None", try to parse Command as full command line
			argsTrimmed := strings.TrimSpace(req.Arguments)
			if len(argsTrimmed) == 0 || argsTrimmed == "None" {
				fields := strings.Fields(strings.TrimSpace(req.Command))
				if len(fields) == 0 {
					c.sendError(message.Token, fmt.Errorf("empty command"))
					return
				}
				command = fields[0]
				if len(fields) > 1 {
					args = fields[1:]
				}
			} else {
				// Normal case: separate command and arguments
				command = strings.TrimSpace(req.Command)
				args = strings.Fields(req.Arguments)
			}

			logVerbosef("Parsed command: '%s', args: %v", command, args)

			cmd := exec.Command(command, args...)
			
			// Configure process attributes for the current platform
			configureProcAttr(cmd)

			stdoutPipe, err := cmd.StdoutPipe()
			if err != nil {
				c.sendError(message.Token, fmt.Errorf("failed to get stdout pipe: %w", err))
				return
			}
			stderrPipe, err := cmd.StderrPipe()
			if err != nil {
				c.sendError(message.Token, fmt.Errorf("failed to get stderr pipe: %w", err))
				return
			}
			stdinPipe, err := cmd.StdinPipe()
			if err != nil {
				c.sendError(message.Token, fmt.Errorf("failed to get stdin pipe: %w", err))
				return
			}

			if err := cmd.Start(); err != nil {
				c.sendError(message.Token, fmt.Errorf("failed to start process: %w", err))
				return
			}

			// Store process in lookup dictionary
			tokenStr := fmt.Sprintf("%x", message.Token)
			c.AddProcess(tokenStr, cmd, stdinPipe)

			// Send continue to indicate process has started
			pidStr := fmt.Sprintf("%d", cmd.Process.Pid)
			c.sendContinue(message.Token, pidStr)

			// Stream stdout / stderr
			go c.streamProcessOutput(message.Token, stdoutPipe, 0)
			go c.streamProcessOutput(message.Token, stderrPipe, 1)

			// Wait for process completion in background
			go func() {
				err := cmd.Wait()
				exitCode := cmd.ProcessState.ExitCode()
				exitCodeStr := fmt.Sprintf("%d", exitCode)
				
				// Clean up process from dictionary
				c.RemoveProcess(tokenStr)
				
				if err != nil {
					c.sendErrorString(message.Token, exitCodeStr)
				} else {
					c.sendOK(message.Token, exitCodeStr)
				}
			}()
		}
	case 402: // Process kill request
		{
			req, err := WSNProcessKillFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			proc, err := os.FindProcess(int(req.PID))
			if err != nil {
				c.sendError(message.Token, fmt.Errorf("failed to find process: %w", err))
				return
			}
			if err := proc.Kill(); err != nil {
				c.sendError(message.Token, fmt.Errorf("failed to kill process: %w", err))
				return
			}
			c.sendOK(message.Token)
		}
	case 404:
		{
			logVerbose("Process STD message received")
			req, err := WSNProcessSTDFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			logVerbose("Request:", req)
			tokenStr := fmt.Sprintf("%x", message.Token)
			processHandle, ok := c.GetProcess(tokenStr)
			if !ok {
				c.sendError(message.Token, fmt.Errorf("process not found for token"))
				return
			}
			logVerbose("Process handle:", processHandle)
			// Write data to process stdin
			if processHandle.stdin != nil {
				_, err := processHandle.stdin.Write([]byte(req.Data))
				if err != nil {
					c.sendError(message.Token, fmt.Errorf("failed to write to process stdin: %w", err))
					return
				}
				//c.sendOK(message.Token)
			} else {
				c.sendError(message.Token, fmt.Errorf("process stdin not available"))
			}
		}
	}
}

func (c *ClientHandler) handleNameResolution(token [16]byte, data []byte) {
	// Parse the incoming struct
	reply, _ := parseAndResolve(data)
	c.sendResolvReply(token, reply)
}

func (c *ClientHandler) sendResolvReply(token [16]byte, data []byte) {
	dataPacket := WSPacket{
		Length:  (uint32)(22 + len(data)),
		CmdType: 202,
		Token:   token,
		Data:    data,
	}

	c.outgoing <- dataPacket
}

func (c *ClientHandler) sendOK(token [16]byte, data ...string) {
	dataStr := ""
	if len(data) > 0 {
		dataStr = data[0]
	}
	okPacket := CreateOKPacket(token, dataStr)
	c.outgoing <- okPacket
}

func (c *ClientHandler) sendError(token [16]byte, err error) {
	byteErr := []byte(err.Error())
	// create buffer with 4-byte length prefix + string data
	buffer := make([]byte, 4+len(byteErr))
	binary.BigEndian.PutUint32(buffer[0:4], uint32(len(byteErr)))
	copy(buffer[4:], byteErr)
	errorPacket := WSPacket{
		Length:  (uint32)(22 + len(buffer)),
		CmdType: 1,
		Token:   token,
		Data:    buffer,
	}

	c.outgoing <- errorPacket
}

func (c *ClientHandler) sendErrorString(token [16]byte, err string) {
	byteErr := []byte(err)
	// create buffer with 4-byte length prefix + string data
	buffer := make([]byte, 4+len(byteErr))
	binary.BigEndian.PutUint32(buffer[0:4], uint32(len(byteErr)))
	copy(buffer[4:], byteErr)
	errorPacket := WSPacket{
		Length:  (uint32)(22 + len(buffer)),
		CmdType: 1,
		Token:   token,
		Data:    buffer,
	}

	c.outgoing <- errorPacket
}

func (c *ClientHandler) sendSocketData(token [16]byte, data []byte) {
	dataPacket := WSPacket{
		Length:  (uint32)(22 + len(data)),
		CmdType: 7,
		Token:   token,
		Data:    data,
	}

	c.outgoing <- dataPacket
}

func (c *ClientHandler) sendServerSocketData(token [16]byte, data []byte) {
	dataPacket := WSPacket{
		Length:  (uint32)(22 + len(data)),
		CmdType: 200,
		Token:   token,
		Data:    data,
	}
	c.outgoing <- dataPacket
}

func (c *ClientHandler) sendContinue(token [16]byte, data ...string) {
	dataStr := ""
	if len(data) > 0 {
		dataStr = data[0]
	}
	continuePacket := CreateContinuePacket(token, dataStr)
	c.outgoing <- continuePacket
}

func (c *ClientHandler) sendGetInfo(token [16]byte) {
	getInfoPacket := CreateGetInfoPacket(token, c.inforeply)
	c.outgoing <- getInfoPacket
}

// Method to handle incoming socket data
func (c *ClientHandler) handleIncomingData(token [16]byte, tokenStr string, conn net.Conn, ch *ClientHandler, connectionTokenOrNil []byte) {
	reader := bufio.NewReader(conn)
	connectionToken := [16]byte{}
	if connectionTokenOrNil != nil {
		connectionToken := make([]byte, 16)
		copy(connectionToken, connectionTokenOrNil)
	}
	for {
		buffer := make([]byte, 65535)
		n, err := reader.Read(buffer)
		if err != nil {
			if connectionTokenOrNil != nil {
				// this belongs to a server socket
				ch.RemoveConnection(tokenStr)
				ch.sendOK(connectionToken)
			} else {
				ch.RemoveConnection(tokenStr)
				ch.sendOK(token)
			}
			break
		}
		receivedData := buffer[:n]
		if connectionTokenOrNil != nil {
			// this belongs to a server socket
			// extract port as string from conn.RemoteAddr()

			remport := conn.RemoteAddr().String()
			remport = remport[strings.LastIndex(remport, ":")+1:]
			remportInt, err := strconv.Atoi(remport)
			if err != nil {
				log.Println("Error converting port to int")
				ch.RemoveConnection(tokenStr)
				ch.sendOK(connectionToken)
				break
			}

			ds := WSNServerSocketData{
				ConnectionToken: connectionToken,
				Data:            receivedData,
				ClientIP:        conn.RemoteAddr().String(),
				ClientPort:      uint16(remportInt),
			}
			data, err := ds.ToData()
			if err != nil {
				log.Printf("Failed to serialize server socket data: %v", err)
				continue
			}
			ch.sendServerSocketData(token, data)

		} else {
			ch.sendSocketData(token, receivedData)
		}
	}
}

func (c *ClientHandler) handleIncomingDataUDP(token [16]byte, tokenStr string, ct [16]byte, udpConn *net.UDPConn, ch *ClientHandler) {
	defer udpConn.Close()

	// We’ll use a loop similar to your TCP case.
	// The difference: use ReadFromUDP to receive data and the remote address.
	buf := make([]byte, 65535) // Adjust size as needed

	for {
		n, addr, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error reading from UDP connection: %v", err)
			ch.RemoveConnection(tokenStr)
			ch.sendOK(token)
			break
		}

		// Slice the buffer to the actual data length
		receivedData := buf[:n]

		ds := WSNServerSocketData{
			ConnectionToken: ct,
			Data:            receivedData,
			ClientIP:        addr.IP.String(),
			ClientPort:      uint16(addr.Port),
		}
		data, err := ds.ToData()
		if err != nil {
			log.Printf("Failed to serialize server socket data: %v", err)
			continue
		}
		ch.sendServerSocketData(token, data)
	}
}

func (c *ClientHandler) handleTCPServer(token [16]byte, tokenStr string, conn net.Listener, ch *ClientHandler) {
	for {
		// Accept a new connection
		newConn, err := conn.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			ch.RemoveConnection(tokenStr)
			ch.sendOK(token)
			break
		}
		// generate random 16 byte token
		connectiontokenRaw := make([]byte, 16)
		_, err = rand.Read(connectiontokenRaw)
		if err != nil {
			log.Fatal(err)
		}
		connectiontoken := [16]byte{}
		copy(connectiontoken[:], connectiontokenRaw)

		connectiontokenStr := fmt.Sprintf("%x", connectiontoken)
		ch.AddConnection(connectiontokenStr, newConn)
		go ch.handleIncomingData(token, tokenStr, newConn, ch, connectiontokenRaw)
		remaddr := newConn.RemoteAddr().String()
		remport := remaddr[strings.LastIndex(remaddr, ":")+1:]
		remportInt, err := strconv.Atoi(remport)
		if err != nil {
			log.Println("Error converting port to int")
			ch.RemoveConnection(tokenStr)
			ch.sendOK(token)
			break
		}

		ds := WSNServerSocketData{
			ConnectionToken: connectiontoken,
			Data:            []byte{},
			ClientIP:        newConn.RemoteAddr().String(),
			ClientPort:      uint16(remportInt),
		}
		data, err := ds.ToData()
		if err != nil {
			log.Printf("Failed to serialize server socket data: %v", err)
			continue
		}
		ch.sendServerSocketData(token, data)
	}
}

func (c *ClientHandler) listDirectory(token [16]byte, dirPath string) {
	// Read directory entries
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		c.sendError(token, err)
		return
	}

	for _, e := range entries {
		// e.Name() is the filename, isDir tells if it's a subdirectory
		name := e.Name()
		isDir := e.IsDir()

		// Build the full path so we can get file info
		fullPath := filepath.Join(dirPath, name)

		// Get additional info (size, mod time) via os.FileInfo
		info, err := e.Info()
		if err != nil {
			log.Printf("Warning: failed to stat %q: %v", fullPath, err)
			continue
		}

		size := info.Size()

		// In Go, we have:
		//   ModTime: info.ModTime()
		// There's no portable standard library method for atime or ctime.
		// If you need them, you must do OS-specific syscalls or store 0.
		modTime := info.ModTime().Unix() // seconds since epoch
		// Let’s set ATime and CTime to 0 for now.
		aTime := modTime
		cTime := modTime

		// Fill the WSNFileEntry fields
		entry := &WSNFileEntry{
			Root:  dirPath,
			Name:  name,
			IsDir: isDir,
			// Size, ATime, MTime, CTime are 8-byte unsigned in the struct
			// so we cast them to uint64.
			Size:  uint64(size),
			ATime: uint64(aTime),
			MTime: uint64(modTime),
			CTime: uint64(cTime),
		}
		data, err := entry.ToData()
		if err != nil {
			log.Printf("Failed to serialize file entry: %v", err)
			continue
		}
		dataPacket := WSPacket{
			Length:  (uint32)(22 + len(data)),
			CmdType: 308,
			Token:   token,
			Data:    data,
		}
		logVerbosef("Sending file entry: %v", entry)
		c.outgoing <- dataPacket
	}

	c.sendOK(token)
}


// listProcesses enumerates running processes (Linux /proc based) and streams WSNProcessInfo packets.
func (c *ClientHandler) listProcesses(token [16]byte) {
	procDir := "/proc"
	entries, err := os.ReadDir(procDir)
	if err != nil {
		c.sendError(token, fmt.Errorf("failed to read /proc: %w", err))
		return
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // skip non-numeric dirs
		}

		// Read cmdline for name/binpath
		cmdlineBytes, _ := os.ReadFile(filepath.Join(procDir, e.Name(), "cmdline"))
		cmdline := strings.TrimRight(string(cmdlineBytes), "\x00")
		parts := strings.Split(cmdline, "\x00")
		procName := ""
		if len(parts) > 0 {
			procName = parts[0]
		}

		info := &WSNProcessInfo{
			PID:             uint64(pid),
			Name:            filepath.Base(procName),
			WindowTitle:     "", // not applicable on Linux CLI
			BinPath:         procName,
			MemoryUsage:     0,
			ThreadCount:     0,
			StartTime:       time.Time{},
			CPUTime:         0,
			IsResponding:    true,
			MainWindowTitle: "",
		}

		data, err := info.ToData()
		if err != nil {
			continue
		}
		// CmdType 403 == PROCINFO
		pkt := WSPacket{
			Length:  uint32(22 + len(data)),
			CmdType: 403,
			Token:   token,
			Data:    data,
		}
		c.outgoing <- pkt
	}

	c.sendOK(token)
}

// streamProcessOutput reads from reader and sends data to client.
func (c *ClientHandler) streamProcessOutput(token [16]byte, r io.ReadCloser, outType uint32) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		std := &WSNProcessSTD{OutType: outType, Data: line}
		data, err := std.ToData()
		if err != nil {
			continue
		}
		// CmdType 404 == PROCSTD
		pkt := WSPacket{
			Length:  uint32(22 + len(data)),
			CmdType: 404,
			Token:   token,
			Data:    data,
		}
		c.outgoing <- pkt
	}
}
