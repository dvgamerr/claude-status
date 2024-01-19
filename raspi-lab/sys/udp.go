package sys

import (
	"fmt"
	"net"
	"time"
)

func PingUDP(host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)

	// Resolve the address to get UDP address
	udpAddr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return err
	}

	// Create a UDP connection
	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Send a simple UDP message
	_, err = conn.Write([]byte("ping"))
	if err != nil {
		return err
	}

	// Set a timeout for receiving a response
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

	// Try to read a response
	buffer := make([]byte, 1024)
	_, err = conn.Read(buffer)
	if err != nil {
		return err
	}

	fmt.Println("UDP Port is open!")
	return nil
}
