

package wsnet
import (
	"bufio"
	"net"
	"fmt"
	"strconv"
	"time"
	"sync"
	"os"
	"path/filepath"
)

type ClientHandler struct {
	send func(WSPacket)
	connections map[string]net.Conn
	fileHandles map[string]*os.File
	connectionsMu sync.Mutex       // Mutex for synchronizing access to connections
	fileHandlesMu sync.Mutex 
	outgoing    chan WSPacket       // Channel for outgoing messages
	inforeply *WSNGetInfoReply

}

// Initialize the ClientHandler struct
func NewClientHandler() *ClientHandler {
	return &ClientHandler{
		connections: make(map[string]net.Conn),
		connectionsMu: sync.Mutex{}, 
		fileHandles: make(map[string]*os.File),
		fileHandlesMu: sync.Mutex{},
		outgoing:    make(chan WSPacket, 1024), // Initialize the outgoing channel
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
	fmt.Println("Client connected")
	c.send = sendfnt
	go c.writePump() // Start the writer goroutine
}

func (c *ClientHandler) OnDisconnect() {
	fmt.Println("Client disconnected")
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

func (ch *ClientHandler) RemoveConnection(key string) {
	ch.connectionsMu.Lock()
	defer ch.connectionsMu.Unlock()
	delete(ch.connections, key)
}

func (ch *ClientHandler) RemoveFileHandle(key string) {
	ch.fileHandlesMu.Lock()
	defer ch.fileHandlesMu.Unlock()
	delete(ch.fileHandles, key)
}


func (c *ClientHandler) OnMessage(message WSPacket) {
	//fmt.Printf("Received: %+v\n", message)
	switch message.CmdType {
	case 0:
		{
			//fmt.Println("OK message received")
			key := fmt.Sprintf("%x", message.Token)
			if _, ok := c.connections[key]; ok {
				c.RemoveConnection(key)
			}

			// if not in connections map, it could be in the future to be in the auth or file map but it's not implemented yet
		}
	case 1:
		{
			//fmt.Println("Error message received")
			key := fmt.Sprintf("%x", message.Token)
			if conn, ok := c.connections[key]; ok {
				conn.Close()
				c.RemoveConnection(fmt.Sprintf("%x", message.Token))
			}
			//same as OK
		}
	case 7:
		{
			//fmt.Println("Socket Data message received")
			key := fmt.Sprintf("%x", message.Token)
			if conn, ok := c.connections[key]; ok {
				conn.Write(message.Data.([]byte))
			}
		}
	case 8:
		{
			//fmt.Println("GetInfo message received")
			// Create a new GetInfo packet
			c.sendGetInfo(message.Token)
		}

	case 5:
		{
			//fmt.Println("Socket Connect message received")
			if connData, ok := message.Data.(*WSNConnect); ok {
				if(connData.Bind) {
					fmt.Println("Bind not supported!")
					return
				} 
				
				fmt.Println("Connecting to", connData.Protocol, "server at IP:", connData.IP, "Port:", connData.Port)
				if (connData.Protocol == "TCP") {
			
					//fmt.Println("TCP Protocol")
					// Connect to the TCP server
					// Create a new TCP connection
					//fmt.Println("Connecting to TCP server at IP:", connData.IP, "Port:", connData.Port)
					timeout := time.Duration(1000 * time.Millisecond)
					conn, err := net.DialTimeout("tcp", connData.IP + ":" + strconv.Itoa(int(connData.Port)), timeout)
					if err != nil {
						//fmt.Println("Error connecting to TCP server:", err)
						c.sendError(message.Token, err)
						return
					}

					tokenStr := fmt.Sprintf("%x", message.Token)
					c.AddConnection(tokenStr, conn)
					//fmt.Println("Connected to TCP server")

					// Start a goroutine to handle incoming data
					go c.handleIncomingData(message.Token, tokenStr, conn, c)
					c.sendContinue(message.Token)

				} else {
					fmt.Println("UDP Protocol")
					//timeout := 1 * time.Second // or similar
					addr := net.JoinHostPort(connData.IP, strconv.Itoa(int(connData.Port)))
					
					fmt.Println("Connecting to UDP server at IP:", addr)
					// net.DialTimeout doesn't apply the same way to UDP, but we can do a Resolve + Dial:
					raddr, err := net.ResolveUDPAddr("udp", addr)
					if err != nil {
						c.sendError(message.Token, err)
						return
					}

					
					conn, err := net.DialUDP("udp", nil, raddr)
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
			} else{
				fmt.Println("Error parsing WSNConnect")
				c.sendError(message.Token, fmt.Errorf("Error parsing WSNConnect"))
			}
		}
	case 200:
		{
			// data to be dispached can be either an UDP client or a SERVER SOCKET
			// servers are not implemented yet

			fmt.Println("Server Socket Data message received")
			connData, err := WSNClientSocketDataFromBytes(message.Data.([]byte))
			if err != nil {
				fmt.Println("Error parsing WSNServerSocketData")
				c.sendError(message.Token, fmt.Errorf("Error parsing WSNServerSocketData"))
				return
			}

			key := fmt.Sprintf("%x", message.Token)
			conn, ok := c.connections[key]
			if !ok {
				fmt.Println("Error getting connection")
				c.sendError(message.Token, fmt.Errorf("Error getting connection"))
				return
			}

			fmt.Println("Sending data to client")
			_, err = conn.Write(connData.Data)
			if err != nil {
				fmt.Println("Error sending data to client")
				c.sendError(message.Token, fmt.Errorf("Error sending data to client"))
				return
			}	
		}

	case 202:
		{
			//fmt.Println("WSNResolv message received")
			go c.handleNameResolution(message.Token, message.Data.([]byte))
		}
	case 300:
		{
			//fmt.Println("List Directory message received")
			// Parse the incoming struct
			req, err := WSNDirLSFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// List the directory
			go c.listDirectory(message.Token, req.Path)
		}
	case 301:
		{
			//create directory
			//fmt.Println("Create Directory message received")
			// Parse the incoming struct
			req, err := WSNDirMKFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Create the directory
			err = os.Mkdir(req.Path, 0755)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 302:
		{
			//remove directory
			//fmt.Println("Remove Directory message received")
			// Parse the incoming struct
			req, err := WSNDirRMFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Remove the directory
			err = os.RemoveAll(req.Path)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
			
		}
	case 303:
		{
			//copy directory
			//fmt.Println("Copy Directory message received")
			// Parse the incoming struct
			req, err := WSNDirCopyFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Copy the directory
			err = CopyDirectory(req.SrcPath, req.DstPath)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)

		}
	case 304:
		{
			//move directory
			//fmt.Println("Move Directory message received")
			// Parse the incoming struct
			req, err := WSNDirMoveFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Move the directory
			err = os.Rename(req.SrcPath, req.DstPath)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 305:
		{
			//fmt.Println("File Open message received")
			// Parse the incoming struct
			req, err := WSNFileOpenFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Open the file
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
			fmt.Println("File Read message received")
			// Parse the incoming struct
			req, err := WSNFileReadFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			fmt.Println("Request:", req)
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
				Length: (uint32)(22 + len(serializedData)),
				CmdType: 307,
				Token: message.Token,
				Data: serializedData,
			}
			c.outgoing <- dataPacket
		}
	case 307:
		{
			//filedata which is used to write file that already is opened
			//fmt.Println("File Write message received")
			// Parse the incoming struct
			req, err := WSNFileDataFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Write the file
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
			//copy file
			//fmt.Println("Copy File message received")
			// Parse the incoming struct
			req, err := WSNFileCopyFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Copy the file
			err = copyFile(req.SrcPath, req.DstPath, nil)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)

		}
	case 310:
		{
			//move file
			//fmt.Println("Move File message received")
			// Parse the incoming struct
			req, err := WSNFileMoveFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Move the file
			err = os.Rename(req.SrcPath, req.DstPath)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 311:
		{
			//remove file
			//fmt.Println("Remove File message received")
			// Parse the incoming struct
			req, err := WSNFileRMFromBytes(message.Data.([]byte))
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			//fmt.Println("Request:", req)
			// Remove the file
			err = os.Remove(req.Path)
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			c.sendOK(message.Token)
		}
	case 312:
		{
			//filestat
			fmt.Println("File Stat message received")
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
			//fmt.Println("File Entry:", entry)
			// Serialize the file entry
			data, err := entry.ToData()
			if err != nil {
				c.sendError(message.Token, err)
				return
			}
			// Send the file entry to the client
			dataPacket := WSPacket{
				Length: (uint32)(22 + len(data)),
				CmdType: 308,
				Token: message.Token,
				Data: data,
			}
			c.outgoing <- dataPacket
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
		Length: (uint32)(22 + len(data)),
		CmdType: 202,
		Token: token,
		Data: data,
	}
	
	c.outgoing <- dataPacket
}

func (c *ClientHandler) sendOK(token [16]byte) {
	okPacket := CreateOKPacket(token)
	c.outgoing <- okPacket
}

func (c *ClientHandler) sendError(token [16]byte, err error) {
	//fmt.Println("Error:", err)
	// Create an error packet
	byteErr := []byte(err.Error())
	errorPacket := WSPacket{
		Length: (uint32)(22 + len(byteErr)),
		CmdType: 1,
		Token: token,
		Data: byteErr,
	}

	c.outgoing <- errorPacket
}


func (c *ClientHandler) sendSocketData(token [16]byte, data []byte) {
	dataPacket := WSPacket{
		Length: (uint32)(22 + len(data)),
		CmdType: 7,
		Token: token,
		Data: data,
	}
	
	c.outgoing <- dataPacket
}

func (c *ClientHandler) sendServerSocketData(token [16]byte, data *WSNServerSocketData) {
	dataPacket := WSPacket{
		Length: (uint32)(22 + len(data.Data)),
		CmdType: 200,
		Token: token,
		Data: data,
	}
	c.outgoing <- dataPacket
}


func (c *ClientHandler) sendContinue(token [16]byte) {
	continuePacket := CreateContinuePacket(token)
	c.outgoing <- continuePacket
}

func (c *ClientHandler) sendGetInfo(token [16]byte) {
	getInfoPacket := CreateGetInfoPacket(token, c.inforeply)
	c.outgoing <- getInfoPacket
}

// Method to handle incoming socket data
func (c *ClientHandler) handleIncomingData(token [16]byte, tokenStr string, conn net.Conn, ch *ClientHandler) {
	reader := bufio.NewReader(conn)
	for {
		// Create a buffer to store the incoming data
		buffer := make([]byte, 65535) // Adjust the buffer size as needed
		n, err := reader.Read(buffer)
		if err != nil {
			//fmt.Println("Error reading from connection:", err)
			// Handle error (e.g., remove connection from map, close connection, etc.)
			ch.RemoveConnection(tokenStr)
			ch.sendOK(token)
			break
		}
		// Process the received bytes
		receivedData := buffer[:n]
		//fmt.Printf("Received data on connection %s: %x\n", tokenStr, receivedData)
		// You can add further processing of the receivedData as needed
		// Send the received data to the client
		ch.sendSocketData(token, receivedData)
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
			fmt.Printf("Error reading from UDP connection: %v\n", err)
            ch.RemoveConnection(tokenStr)
            ch.sendOK(token)
            break
        }

        // Slice the buffer to the actual data length
        receivedData := buf[:n]

        ds := &WSNServerSocketData{
			ConnectionToken : ct,
			Data : receivedData,
			ClientIP : addr.IP.String(),
			ClientPort : uint16(addr.Port),
		}
        ch.sendServerSocketData(token, ds)
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
            // If we can't get info for this entry, skip or handle error as you prefer
            fmt.Printf("Warning: failed to stat %q: %v\n", fullPath, err)
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
			fmt.Printf("Failed to serialize file entry: %v", err)
			continue
		}
        dataPacket := WSPacket{
			Length: (uint32)(22 + len(data)),
			CmdType: 308,
			Token: token,
			Data: data,
		}
		fmt.Printf("Sending file entry: %v\n", entry)
		c.outgoing <- dataPacket
    }

	c.sendOK(token)
}

