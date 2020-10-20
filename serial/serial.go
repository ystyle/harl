package serial

import (
	"fmt"
	"github.com/tarm/serial"
	"io"
	"log"
	"os"
	"strings"
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
	go handle.read()
	return handle
}

func (s *Serial) read() {
	buffer := make([]byte, 1024)
	for {
		n, err := s.port.Read(buffer)
		if err == io.EOF {
			continue
		}
		serverMsg := string(buffer[0:n])
		fmt.Print(serverMsg)
		if !s.started && strings.HasSuffix(serverMsg, "OHOS # ") {
			s.started = true
			fmt.Println("serial: connected.")
			return
		}
		if serverMsg == "bye" {
			os.Exit(0)
		}
	}
}

func (s *Serial) Send(msg string) {
	fmt.Println("执行命令: ", msg)
	s.port.Write([]byte(fmt.Sprintf("%s \n", msg)))
	buffer := make([]byte, 1024)
	for {
		n, err := s.port.Read(buffer)
		if err == io.EOF {
			fmt.Print("\n")
			break
		}
		serverMsg := string(buffer[0:n])
		//fmt.Print(serverMsg)
		if strings.Contains(serverMsg, "OHOS # ") {
			break
		}
		if n < 1024 {
			break
		}
	}
	time.Sleep(time.Millisecond * 500)
	fmt.Println("-------------------------------------------")
}
