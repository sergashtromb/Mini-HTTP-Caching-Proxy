package tools

import "net"

func SendSuccesConnection(conn net.Conn) error {
	_, err := conn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	return err
}

func SendBadGatterway(conn net.Conn) error {
	_, err := conn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
	return err
}