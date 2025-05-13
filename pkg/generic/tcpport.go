package generic

import (
	"net"
	"strconv"
	"time"
)

const (
	StartPort = 32100
)

func GetFreePort(begin int) int {
	for port := begin; port < 32767; port++ {
		sp := strconv.Itoa(port)
		if err := ping(sp); err != nil {
			return port
		}
	}
	return -1
}

func ping(port string) error {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("", port), 5*time.Second)
	if conn != nil {
		defer conn.Close()
	}
	return err
}
