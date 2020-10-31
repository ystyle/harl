package main

import (
	"bufio"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
	"harl/model"
	"harl/serial"
	"harl/utils"
	"harl/watcher"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

var (
	s           *serial.Serial
	config      model.Config
	bundleName  string
	abilityName string
)

func main() {
	app := &cli.App{
		Name:  "harl",
		Usage: "Open Harmony OS Dev tools",
		Before: func(c *cli.Context) error {
			if c.Args().Len() == 0 {
				return nil
			}
			switch c.Args().First() {
			case "init":
				setAppInfo()
				return nil
			case "watch", "w":
				setAppInfo()
			}
			config = model.ReadConfig()
			s = serial.New(config.Shell.Com)
			s.IsConnected()
			return nil
		},
		Commands: []*cli.Command{
			Init(),
			watch(),
			install(),
			uninstall(),
			shell(),
			reboot(),
		},
		After: func(c *cli.Context) error {
			if s != nil {
				s.Close()
			}
			return nil
		},
	}
	app.Version = "v0.2.1"
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func sendAndRead(msg string) {
	fmt.Println("command>", msg)
	s.Send(msg)
	out := s.Read()
	if out != "" {
		fmt.Print(out)
	}
}

func send(msg string) {
	fmt.Println("command>", msg)
	s.Send(msg)
}

func setAppInfo() error {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	bs, err := ioutil.ReadFile(path.Join(pwd, "entry/src/main/config.json"))
	if err != nil {
		return err
	}
	bundle, err := jsonparser.GetString(bs, "app", "bundleName")
	if err != nil {
		return err
	}
	ability, err := jsonparser.GetString(bs, "module", "abilities", "[0]", "name")
	if err != nil {
		return err
	}
	bundleName = bundle
	abilityName = ability
	return nil
}

func Init() *cli.Command {
	return &cli.Command{
		Name:    "init",
		Aliases: []string{"i"},
		Usage:   "init .harm.yml",
		Action: func(c *cli.Context) error {
			var config = model.Config{
				Watch: &model.Watch{
					//Project:   pwd,
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
					Delay: 100,
				},
				Nfs: &model.Nfs{
					Ldir: "H:/bin",
					Rdir: "/nfs",
				},
				Shell: &model.Shell{
					Com: "COM5",
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
			fmt.Println("start watch")
			w := watcher.New(config.Watch)
			go w.Watcher()
			go func() {
				for {
					out := s.Read()
					if out != "" {
						fmt.Print(out)
					}
				}
			}()
			go ReadLineAndExec()
			for {
				select {
				case event := <-w.Event:
					switch event.Action {
					case "build":
						fmt.Println("build ...")
						err := utils.Run(path.Join(config.Watch.Project, "gradlew.bat"), "assembleDebug")
						if err != nil {
							fmt.Println("build failed.")
						} else {
							fmt.Println("build finished.")
						}
					case "reload":
						fmt.Println("reload...")
						t := time.Now().Format("20060102-150405")
						// copy to nfs dir
						fmt.Println("copy file to nfs ...")
						form := event.Data["bin"]

						hap := fmt.Sprintf("%s-%s.hap", bundleName, t)
						utils.Copy(form, path.Join(config.Nfs.Ldir, hap))
						// install
						fmt.Println("install...")
						send(fmt.Sprintf("cd %s", config.Nfs.Rdir))
						send(fmt.Sprintf("./bm set -s disable"))
						send(fmt.Sprintf("./bm set -d enable"))
						time.Sleep(time.Second * 1)
						send(fmt.Sprintf("./bm install -p %s", hap))
						// start app
						fmt.Println("start...")
						time.Sleep(time.Second * 3)
						send(fmt.Sprintf("./aa start -p %s -n %s", bundleName, abilityName))
					default:
						fmt.Println("unknow operation", event)
					}
				}
			}
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
				sendAndRead(commang)
				return nil
			}

			fmt.Println("input quit to exit")
			go func() {
				for {
					out := s.Read()
					if out != "" {
						fmt.Print(out)
					}
				}
			}()
			ReadLineAndExec()
			return nil
		},
	}
}

func ReadLineAndExec() {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, _, _ := reader.ReadLine()
		msg := strings.TrimSpace(string(line))
		if msg == "quit" {
			os.Exit(0)
		}
		if strings.HasPrefix(msg, "^run ") {
			commandType := strings.Replace(msg, "^run ", "", -1)
			commandType = strings.TrimSpace(commandType)
			if commands, ok := config.Command[commandType]; ok {
				for _, command := range commands {
					send(command)
					time.Sleep(time.Second)
				}
			}
			continue
		}
		s.Send(msg)
	}
}

func reboot() *cli.Command {
	return &cli.Command{
		Name:  "reboot",
		Usage: "reboot",
		Action: func(c *cli.Context) error {
			send("reset")
			return nil
		},
	}
}

func install() *cli.Command {
	return &cli.Command{
		Name:  "install",
		Usage: "install hap",
		Action: func(c *cli.Context) error {
			form := c.Args().First()
			t := time.Now().Format("20060102-150405")
			hap := fmt.Sprintf("hap-%s.hap", t)
			lhap := path.Join(config.Nfs.Ldir, hap)
			_, err := utils.Copy(form, lhap)
			if err != nil {
				return err
			}
			fmt.Println("install...")
			send(fmt.Sprintf("cd %s", config.Nfs.Rdir))
			send(fmt.Sprintf("./bm set -s disable"))
			send(fmt.Sprintf("./bm set -d enable"))
			time.Sleep(time.Second * 1)
			send(fmt.Sprintf("./bm install -p %s", hap))
			return nil
		},
	}
}

func uninstall() *cli.Command {
	return &cli.Command{
		Name:  "uninstall",
		Usage: "uninstall hap",
		Action: func(c *cli.Context) error {
			cmd := fmt.Sprintf("cd %s", config.Nfs.Rdir)
			send(cmd)
			send(fmt.Sprintf("./bm uninstall -n %s", c.Args().First()))
			return nil
		},
	}
}
