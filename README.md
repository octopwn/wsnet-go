# WSNet Implementation in Golang

## Overview
This project, referred to as the "proxy" in the documentation, is designed to provide the necessary WebSocket-to-TCP translation required for the tools running inside of OctoPwn. This enables network communication between various components.

## How It Works
Upon starting the application, a WebSocket server is set up on `localhost` at port `8700`. The server then waits for a connection from OctoPwn to facilitate the required communication.

## Features
- **WebSocket-to-TCP translation**: Facilitates communication between OctoPwn and other networked components.

## Support Matrix

| Feature                 | Windows | Linux | Mac |
|-------------------------|---------|-------|-----|
| TCP Client              | ✔       | ✔     | ✔   |
| TCP Server              | ✘       | ✘     | ✘   |
| UDP Client              | ✔       | ✔     | ✔   |
| UDP Server              | ✘       | ✘     | ✘   |
| Local File Browser      | ✔       | ✔     | ✔   |
| Authentication Proxy    | ✔       | ✔     | ✔   |

## Getting Started
### Prerequisites
- Golang version 1.22.5 or later
- [OctoPwn](https://live.octopwn.com) set up and running

### Installation
Clone the repository and navigate into the project directory:

```bash
git clone https://github.com/yourusername/wsnet-proxy.git
cd wsnet-proxy
# for linux OS
GOOS=linux GOARCH=amd64 go build -o wsnet-go
# for windows OS (supports auth proxy)
GOOS=windows GOARCH=amd64 go build -o wsnet-go.exe
# for tailscale version - linux -
GOOS=linux GOARCH=amd64 go build -tags tailscale -o wsnet_tailscale
# for tailscale version - windows -
GOOS=windows GOARCH=amd64 go build -tags tailscale -o wsnet_tailscale.exe
```

# Usage
Run the proxy server with the following command:
```bash
./wsnet-go [options]
```

## Tailscale
First time running you must provide a TS_AUTH_KEY as an env variable. Subsequent runs do not need the key.
```bash
TS_AUTH_KEY=tskey-auth-xx... ./wsnet_tailscale
```

### Command-Line Options

The following command-line options are available:

- **`-port`**: Specifies the port on which the WebSocket server will listen. The default is `8700`.
  - Example: `-port 8080` will start the server on port 8080.
  
- **`-address`**: Defines the address to which the server will bind. By default, it binds to `localhost`.
  - Example: `-address 0.0.0.0` will bind the server to all available network interfaces.
  
- **`-uri-path`**: Sets the URI path (or UUID) for the WebSocket connection. This can be used to create specific endpoints for WebSocket communication.
  - Example: `-uri-path /ws/connection` will make the WebSocket server available at `ws://localhost:8700/ws/connection`.
  
- **`-ssl`**: Enables TLS security for the WebSocket connection.
  - Example: `-ssl` will start the WebSocket server with TLS encryption.

# Limitations
The application currently only supports the TCP client functionality. Other features such as TCP server, UDP client/server, local file browsing, and authentication proxy are not yet implemented.

# Roadmap

- [ ] **TCP Server Support**: Enable the application to function as a TCP server, allowing it to accept incoming TCP connections.
- [ ] **UDP Server**: Add support for UDP server functionalities to handle datagram-based communication.


# Contributing
Contributions are welcome! Please submit a pull request or open an issue if you have suggestions or find a bug.