package serial

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/tarm/serial"
	"io"
	"log"
	"time"
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

func (s *Serial) Send(msg string) {
	command := fmt.Sprintf("%s \n", msg)
	s.port.Write([]byte(command))
}

func (s *Serial) Read() string {
	buffer := make([]byte, 1024)
	var buff bytes.Buffer
	for {
		time.Sleep(time.Millisecond * 100)
		n, err := s.port.Read(buffer)
		if err == io.EOF {
			break
		}
		buff.Write(buffer[:n])
		if n == 1024 {
			continue
		}
		if n < 1024 {
			break
		}
	}

	return buff.String()
}

func (s *Serial) Close() {
	s.port.Close()
}
