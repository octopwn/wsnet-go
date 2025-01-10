package wsnet

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "net"
    "sync"
	"log"
	"time"
	"context"
)

// WSNResolv is the Go equivalent of the Python WSNResolv class.
type WSNResolv struct {
    // Number of host/IP strings
    Count uint32

    // The list of host/IP strings
    IPOrHostnames []string
}


// FromBytes parses the raw data in the same format as the Python WSNResolv.from_bytes().
func WSNResolvFromBytes(b []byte) (*WSNResolv, error) {
    r := bytes.NewReader(b)

    var wr WSNResolv
    // 3) Read 4-byte count
    if err := binary.Read(r, binary.BigEndian, &wr.Count); err != nil {
        return nil, fmt.Errorf("failed to read count: %w", err)
    }

    // 4) Read each hostname string
    wr.IPOrHostnames = make([]string, 0, wr.Count)
    for i := uint32(0); i < wr.Count; i++ {
        s, err := readStr(r)
        if err != nil {
            return nil, fmt.Errorf("failed to read string #%d: %w", i, err)
        }
        wr.IPOrHostnames = append(wr.IPOrHostnames, s)
    }

    return &wr, nil
}


// ToData produces the byte slice in the same format as the Python WSNResolv.to_data().
func (wr *WSNResolv) ToData() ([]byte, error) {
    var buf bytes.Buffer
    
    // 3) Write count (4 bytes, big-endian)
    //    Just in case, ensure wr.Count matches wr.IPOrHostnames length
    wr.Count = uint32(len(wr.IPOrHostnames))
    if err := binary.Write(&buf, binary.BigEndian, wr.Count); err != nil {
        return nil, fmt.Errorf("failed to write count: %w", err)
    }

    // 4) For each hostname string, write length + data
    for _, host := range wr.IPOrHostnames {
        if err := writeStr(&buf, host); err != nil {
            return nil, fmt.Errorf("failed to write string: %w", err)
        }
    }

    return buf.Bytes(), nil
}


// parseAndResolve uses either LookupAddr for IP addresses or LookupHost for hostnames,
// with concurrency limiting (20 at a time) and a context timeout (2 seconds).
func parseAndResolve(b []byte) ([]byte, error) {
    // 1) Parse the incoming struct
    req, err := WSNResolvFromBytes(b)
    if err != nil {
        return nil, fmt.Errorf("failed to parse WSNResolv: %w", err)
    }

    // Prepare result slice
    results := make([]string, len(req.IPOrHostnames))

    // WaitGroup for concurrency
    var wg sync.WaitGroup

    // Semaphore channel to limit concurrency to 20
    sem := make(chan struct{}, 20)

    // Define the DNS resolution timeout
    lookupTimeout := 2 * time.Second

    for i, addr := range req.IPOrHostnames {
        wg.Add(1)
        sem <- struct{}{} // acquire a slot

        i, addr := i, addr // capture loop variables
        go func() {
            defer wg.Done()
            defer func() { <-sem }() // release slot

            // Create a context for each lookup
            ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
            defer cancel()

            // Check if the string is a valid IP
            ip := net.ParseIP(addr)
            if ip != nil {
                // It's an IP -> do a reverse DNS lookup
                names, err := net.DefaultResolver.LookupAddr(ctx, addr)
                if err != nil || len(names) == 0 {
                    log.Printf("Reverse lookup failed for IP %q: %v\n", addr, err)
                    results[i] = ""
                } else {
                    // Typically net.LookupAddr returns FQDNs, e.g. ["example.com."]
                    // We'll store the first result. Optionally trim trailing dot.
                    results[i] = names[0]
                }
            } else {
                // It's a hostname -> do a forward lookup
                ips, err := net.DefaultResolver.LookupHost(ctx, addr)
                if err != nil || len(ips) == 0 {
                    log.Printf("Forward lookup failed for host %q: %v\n", addr, err)
                    results[i] = ""
                } else {
                    results[i] = ips[0] // pick the first IP
                }
            }
        }()
    }

    // Wait until all goroutines finish
    wg.Wait()

    // Build the reply WSNResolv
    reply := &WSNResolv{
        IPOrHostnames: results,
    }

    // Serialize the reply
    return reply.ToData()
}