package main

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
	"harl/model"
	"harl/serial"
	"harl/utils"
	"harl/watcher"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

var (
	ws   *websocket.Conn // websocket connect
	ldir string          // local nfs mounted dir
	rdir string          // remote nfs mounted dir
)

func main() {
	app := &cli.App{
		Name:  "harl",
		Usage: "Open Harmony OS Dev tools",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "serial",
				Aliases: []string{"s"},
				Usage:   "serial port",
			},
		},
		Before: func(c *cli.Context) error {
			switch c.Args().First() {
			case "init", "i", "daemon", "connect":
				return nil
			case "disconnect", "reboot", "shell", "watch", "w", "uninstall", "install":
				// connect ws
				fmt.Println(c.Args().First())
				u := url.URL{Scheme: "ws", Host: "localhost:1614", Path: "/"}
				conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
				if err != nil {
					panic(err)
				}
				ws = conn
				// long connect
				switch c.Args().First() {
				case "shell", "watch", "w":
					go Read()
				}
				switch c.Args().First() {
				case "install", "uninstall":
					SendAndRead("[config]")
				}
			default:
				return nil
			}
			return nil
		},
		Commands: []*cli.Command{
			Init(),
			connect(),
			disconnect(),
			watch(),
			install(),
			uninstall(),
			shell(),
			reboot(),
			daemon(),
		},
		After: func(context *cli.Context) error {
			if ws != nil {
				ws.Close()
			}
			return nil
		},
	}
	app.Version = "v0.1.1"
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func Send(msg string) {
	ws.WriteMessage(websocket.TextMessage, []byte(msg))
}

func SendAndRead(msg string) {
	ws.WriteMessage(websocket.TextMessage, []byte(msg))
	readLine()
}

func Read() {
	for {
		readLine()
	}
}

func readLine() {
	_, message, err := ws.ReadMessage()
	if err != nil {
		fmt.Println(err)
		return
	}
	msg := string(message)
	if strings.HasPrefix(msg, "config:") {
		fmt.Println("config: ", msg)
		config := strings.ReplaceAll(msg, "config:", "")
		dirs := strings.Split(config, "|")
		fmt.Println(dirs)
		ldir = dirs[0]
		rdir = dirs[1]
		fmt.Println(ldir, rdir)
	} else {
		fmt.Printf("%s", message)
	}
}

func Init() *cli.Command {
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "init .harm.yml",
		Action: func(c *cli.Context) error {
			pwd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			bs, err := ioutil.ReadFile(path.Join(pwd, "entry/src/main/config.json"))
			if err != nil {
				panic(err)
			}
			app, err := jsonparser.GetString(bs, "app", "bundleName")
			if err != nil {
				panic(err)
			}
			deviceType, err := jsonparser.GetString(bs, "module", "deviceType", "[0]")
			if err != nil {
				panic(err)
			}
			abilityName, err := jsonparser.GetString(bs, "module", "abilities", "[0]", "name")
			if err != nil {
				panic(err)
			}
			var config = model.Config{
				Build: &model.Build{
					//Project:   pwd,
					BuildType: deviceType,
					Excludes: []string{
						".gradle",
						".idea",
						"gradle",
						"entry/build",
						"entry/node_modules",
					},
					Includes: []string{
						".css",
						".hml",
						".js",
						".hap",
						".json",
					},
					Nfsdir: "H:/bin",
					Delay:  100,
				},
				Reload: &model.Reload{
					Dir: "/nfs",
					//Telnet:      "192.168.3.10",
					Com:         "COM5",
					BundleName:  app,
					AbilityName: abilityName,
				},
			}
			data, err := yaml.Marshal(config)
			if err != nil {
				panic(err)
			}
			err = ioutil.WriteFile(".harl.yaml", data, 0660)
			if err != nil {
				panic(err)
			}
			return nil
		},
	}
}

func watch() *cli.Command {
	return &cli.Command{
		Name:    "watch",
		Aliases: []string{"w"},
		Usage:   "watch and reload app",
		Action: func(c *cli.Context) error {
			config := model.ReadConfig()
			w := watcher.New(config)
			go w.Watcher()
			for {
				select {
				case event := <-w.Event:
					switch event.Action {
					case "build":
						fmt.Println("build ...")
						err := utils.Run(path.Join(config.Build.Project, "gradlew.bat"), "assembleDebug")
						if err != nil {
							fmt.Println("build failed.")
						} else {
							fmt.Println("build finished.")
						}
					case "reload":
						fmt.Println("reload...")
						t := time.Now().Format("20060102-150405")
						buildType := config.Build.BuildType
						// copy to nfs dir
						fmt.Println("copy file to nfs ...")
						form := fmt.Sprintf("build/outputs/hap/debug/%s/entry-debug-%s-unsigned.hap", buildType, buildType)
						hap := fmt.Sprintf("%s-%s.hap", config.Reload.BundleName, t)
						utils.Copy(path.Join(config.Build.Project, form), path.Join(config.Build.Nfsdir, hap))
						// install
						fmt.Println("install...")
						Send(fmt.Sprintf("cd %s", config.Reload.Dir))
						Send(fmt.Sprintf("./bm set -s disable"))
						Send(fmt.Sprintf("./bm set -d enable"))
						Send(fmt.Sprintf("./bm install -p %s", hap))

						// start app
						fmt.Println("start...")
						Send(fmt.Sprintf("./aa start -p %s -n %s", config.Reload.BundleName, config.Reload.AbilityName))
					default:
						fmt.Println("unknow operation", event)
					}
				}
			}
			return nil
		},
	}
}

func connect() *cli.Command {
	return &cli.Command{
		Name:  "connect",
		Usage: "conect with serial",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "serial",
				Aliases: []string{"s"},
				Usage:   "serial port",
			},
			&cli.StringFlag{
				Name:    "local",
				Aliases: []string{"ldir", "l"},
				Usage:   "local nfs mounted dir",
			},
			&cli.StringFlag{
				Name:    "remote",
				Aliases: []string{"rdir", "r"},
				Usage:   "remote nfs mounted dir",
			},
		},
		Action: func(c *cli.Context) error {
			// config form file
			s := c.String("serial")
			l := c.String("local")
			r := c.String("remote")
			if exists, _ := utils.IsExists(".harl.yaml"); exists {
				config := model.ReadConfig()
				s = config.Reload.Com
				l = config.Build.Nfsdir
				r = config.Reload.Dir
			}
			// config form args
			if s == "" || l == "" || r == "" {
				return errors.New("command needs enough arguments")
			}
			args := []string{
				"daemon",
				"-s", s,
				"-l", l,
				"-r", r,
			}
			cmd := exec.Command(os.Args[0], args...)
			return cmd.Start()
		},
	}
}

func daemon() *cli.Command {
	return &cli.Command{
		Name:  "daemon",
		Usage: "run harl daemon",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "serial",
				Aliases:  []string{"s"},
				Usage:    "serial port",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "local",
				Aliases:  []string{"ldir", "l"},
				Usage:    "local nfs mounted dir",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "remote",
				Aliases:  []string{"rdir", "r"},
				Usage:    "remote nfs mounted dir",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			ldir = c.String("local")
			rdir = c.String("remote")
			s := serial.New(c.String("serial"))
			err := s.IsConnected()
			if err != nil {
				return err
			}
			fmt.Println("connected.")
			var upgrader = websocket.Upgrader{} // use default options
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				c, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					log.Print("upgrade:", err)
					return
				}
				defer c.Close()
				go func() {
					for {
						bs, err := s.Read()
						if err != nil {
							continue
						}
						fmt.Printf("%s", bs)
						err = c.WriteMessage(websocket.TextMessage, bs)
						if err != nil {
							fmt.Print(err)
						}
						time.Sleep(time.Millisecond * 200)
					}
				}()
				for {
					_, message, err := c.ReadMessage()
					if err != nil {
						log.Println("read:", err)
						break
					}
					log.Printf("recv: %s", message)
					if string(message) == "[config]" {
						config := fmt.Sprintf("config:%s|%s", ldir, rdir)
						err = c.WriteMessage(websocket.TextMessage, []byte(config))
						if err != nil {
							log.Println("read:", err)
							return
						}
						continue
					}
					if string(message) == "[ws close]" {
						os.Exit(0)
					}
					s.Send(message)
				}
			})
			log.Fatal(http.ListenAndServe(":1614", nil))
			return nil
		},
	}
}

func shell() *cli.Command {
	return &cli.Command{
		Name:  "shell",
		Usage: "open a shell",
		Action: func(c *cli.Context) error {
			if c.Args().Len() > 0 {
				commang := strings.Join(c.Args().Slice(), " ")
				Send(commang)
				return nil
			}
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("input quit to exit")
			for {
				line, _, _ := reader.ReadLine()
				msg := string(line)
				if msg == "quit" {
					os.Exit(0)
				}
				Send(msg)
			}
			return nil
		},
	}
}

func disconnect() *cli.Command {
	return &cli.Command{
		Name:  "disconnect",
		Usage: "close daemon sever",
		Action: func(c *cli.Context) error {
			ws.WriteMessage(websocket.TextMessage, []byte("[ws close]"))
			return nil
		},
	}
}

func reboot() *cli.Command {
	return &cli.Command{
		Name:  "reboot",
		Usage: "reboot",
		Action: func(c *cli.Context) error {
			ws.WriteMessage(websocket.TextMessage, []byte("reset"))
			return nil
		},
	}
}

func install() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "install hap",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "path",
				Aliases:  []string{"p"},
				Usage:    "hap path",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			form := c.Path("path")
			fmt.Println(form)
			t := time.Now().Format("20060102-150405")
			hap := fmt.Sprintf("hap-%s.hap", t)
			lhap := path.Join(ldir, hap)
			_, err := utils.Copy(form, lhap)
			if err != nil {
				return err
			}
			time.Sleep(time.Second * 1)
			fmt.Println("install...")
			SendAndRead(fmt.Sprintf("cd %s", rdir))
			SendAndRead(fmt.Sprintf("./bm set -s disable"))
			SendAndRead(fmt.Sprintf("./bm set -d enable"))
			SendAndRead(fmt.Sprintf("./bm install -p %s", hap))
			return nil
		},
	}
}

func uninstall() *cli.Command {
	return &cli.Command{
		Name:  "uninstall",
		Usage: "uninstall hap",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:     "bundlename",
				Aliases:  []string{"n"},
				Usage:    "hap bundlename",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			cmd := fmt.Sprintf("cd %s", rdir)
			SendAndRead(cmd)
			SendAndRead(fmt.Sprintf("./bm uninstall -n %s", c.String("bundlename")))
			return nil
		},
	}
}
