package tools

import (
	"encoding/binary"
	"net"
	"sync"
)
const (
    HOSTNAME = "server" 
    PORT = "8080"
    SERVER_TYPE = "tcp"
    PATH = HOSTNAME+":"+PORT
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

// Thread que lida com o recebimento de mensagens, manda os dados recebidos para o canal received_data
func ReceiveHandler(conn net.Conn, received_data chan []byte, wg *sync.WaitGroup, error_chan chan error) {
	header := make([]byte, 4)
	for {
		err := receiveMessage(conn, header)
		if err != nil {
      error_chan <- err
			wg.Done()
			return
		}
		data_length := int(binary.BigEndian.Uint32(header))
		data := make([]byte, data_length)
		err = receiveMessage(conn, data)
		if err != nil {
      error_chan <- err
			wg.Done()
			return
		}
		received_data <- data
	}
}

// Thread que lida com o envio de mensagens pela conexão estabelecida
func SendHandler(conn net.Conn, send_data chan []byte, wg *sync.WaitGroup, error_chan chan error) {
	for {
		data := <-send_data
		data_size := uint32(len(data))
		header := make([]byte, 4)
		binary.BigEndian.PutUint32(header, data_size)
		_, err := conn.Write(header)
		if err != nil {
      error_chan <- err
			wg.Done()
			return
		}
		_, err = conn.Write(data)
		if err != nil {
      error_chan <- err
			wg.Done()
			return
		}
	}
}
