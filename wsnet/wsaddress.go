package wsnet

import (
	"fmt"
	"net"
	"strings"
)

// Function to retrieve all non-loopback IP addresses
func getLocalIPAddresses() ([]string, error) {
	var ips []string

	// Retrieve a list of all network interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %v", err)
	}

	for _, iface := range ifaces {
		// Skip interfaces that are down or loopback
		if iface.Flags&net.FlagUp == 0 {
			continue // interface is down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}

		// Retrieve addresses for the interface
		addrs, err := iface.Addrs()
		if err != nil {
			fmt.Printf("warning: unable to get addresses for interface %s: %v\n", iface.Name, err)
			continue
		}

		for _, addr := range addrs {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			// Skip IPv6 link-local and unspecified addresses
			if ip == nil || ip.IsLoopback() {
				continue
			}

			// Convert IPv4-mapped IPv6 addresses to IPv4
			ip = ip.To4()
			if ip == nil {
				continue // not an IPv4 address
			}

			ips = append(ips, ip.String())
		}
	}

	return ips, nil
}

// Function to construct and print URLs based on address and path
func printAccessibleURLs(address, path string, port string, ssl bool) error {
	// Normalize the path to ensure it starts with "/"
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	protocol := "ws"
	if ssl {
		protocol = "wss"
	}

	// Check if the address is "0.0.0.0"
	if address == "0.0.0.0" {
		// Retrieve all local IP addresses
		ips, err := getLocalIPAddresses()
		if err != nil {
			// disregard error, continue with address "
			fmt.Printf("warning: failed to get local IP addresses: %v\n", err)
			ips = []string{address}
		}

		if len(ips) == 0 {
			fmt.Println("No valid IP addresses found.")
			return nil
		}

		// Construct and print URLs for each IP
		fmt.Println("Accessible URLs:")
		for _, ip := range ips {
			if port != "" {
				fmt.Printf("%s://%s:%s%s\n", protocol, ip, port, path)
			} else {
				fmt.Printf("%s://%s%s\n", protocol, ip, path)
			}
		}
	} else {
		// Address is not "0.0.0.0", construct URL directly
		// Optionally, validate if the address is a valid IP or hostname
		if port != "" {
			fmt.Printf("Accessible URL: %s://%s:%s%s\n", protocol, address, port, path)
		} else {
			fmt.Printf("Accessible URL: %s://%s%s\n", protocol, address, path)
		}
	}

	return nil
}
