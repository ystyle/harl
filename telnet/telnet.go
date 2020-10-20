package telnet

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

type Telnet struct {
	Conn    net.Conn
	started bool
}

func ClientHandleError(err error, when string) {
	if err != nil {
		fmt.Println(err, when)
		os.Exit(1)
	}
}

func New(ip string) *Telnet {
	// 拨号远程地址，简历tcp连接
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:23", ip))
	ClientHandleError(err, "client conn error")
	fmt.Printf("telnet: connecting to [%s:23] ...\n", ip)
	tel := &Telnet{
		Conn: conn,
	}
	go tel.read()
	return tel
}

func (tel *Telnet) read() {
	buffer := make([]byte, 1024)
	for {
		n, err := tel.Conn.Read(buffer)
		if err == io.EOF {
			continue
		}
		serverMsg := string(buffer[0:n])
		fmt.Print(serverMsg)
		if !tel.started && strings.HasSuffix(serverMsg, "OHOS # ") {
			tel.started = true
			fmt.Println("telnet: connected.")
			return
		}
		if serverMsg == "bye" {
			os.Exit(0)
		}
	}
}

func (tel *Telnet) Send(msg string) {
	fmt.Println("执行命令: ", msg)
	tel.Conn.Write([]byte(fmt.Sprintf("%s \n", msg)))
	buffer := make([]byte, 1024)
	for {
		n, err := tel.Conn.Read(buffer)
		if err == io.EOF {
			fmt.Print("\n")
			break
		}
		serverMsg := string(buffer[0:n])
		fmt.Print(serverMsg)
		if strings.HasSuffix(serverMsg, "OHOS # ") {
			break
		}
	}
	fmt.Println("-------------------------------------------")
}
