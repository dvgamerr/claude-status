package sys

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

func PingIP(ip string) error {
	conn, err := net.DialTimeout("ip4:icmp", ip, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil
}

func PingTCP(host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, 500*time.Millisecond)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func TCPVerifySSL(host string, port int) error {
	address := fmt.Sprintf("%s:%d", host, port)

	// Connect to the server
	conn, err := tls.Dial("tcp", address, &tls.Config{
		InsecureSkipVerify: true, // Skip SSL verification (change to false for production)
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	// Verify the server's certificate
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("no certificates found")
	}

	// Check the certificate verification result
	if state.VerifiedChains == nil || len(state.VerifiedChains) == 0 {
		return fmt.Errorf("certificate verification failed")
	}

	fmt.Println("Server's Common Name (CN):", state.PeerCertificates[0].Subject.CommonName)

	return nil
}
