package protocol

import (
	"encoding/binary"
	"net"
	"sync"
)

// Lê da conecção até completar o tamanho do buffer
func receiveMessage(conn net.Conn, buffer []byte) error {
	bytes_received := 0
	for bytes_received < len(buffer) {
		readed, err := conn.Read(buffer[bytes_received:])
		if err != nil {
			return err
		}
		bytes_received += readed
	}
	return nil
}
