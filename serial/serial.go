package serial

import (
	"errors"
	"fmt"
	"github.com/tarm/serial"
	"io"
	"log"
)

type Serial struct {
	port    *serial.Port
	started bool
}

func New(com string) *Serial {
	fmt.Printf("Serial: connecting to [%s] ...\n", com)
	port := &serial.Config{Name: "COM5", Baud: 115200}
	s, err := serial.OpenPort(port)
	if err != nil {
		log.Fatal(err)
	}
	handle := &Serial{
		port: s,
	}
	return handle
}

func (s *Serial) IsConnected() error {
	s.port.Write([]byte("\n"))
	buffer := make([]byte, 1024)
	n, err := s.port.Read(buffer)
	if err == io.EOF {
		return err
	}
	serverMsg := string(buffer[0:n])
	if serverMsg == "bye" {
		return errors.New("connection failed")
	}
	return nil
}

func (s *Serial) Send(msg []byte) {
	command := fmt.Sprintf("%s \n", msg)
	s.port.Write([]byte(command))
}

func (s *Serial) Read() ([]byte, error) {
	buffer := make([]byte, 1024)
	n, err := s.port.Read(buffer)
	if err == io.EOF {
		return nil, err
	}
	return buffer[0:n], nil
}
