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
	s      *serial.Serial
	config model.Config
)

func main() {
	app := &cli.App{
		Name:  "harl",
		Usage: "Open Harmony OS Dev tools",
		Before: func(c *cli.Context) error {
			if c.Args().First() == "init" || c.Args().Len() == 0 {
				return nil
			}
			config = model.ReadConfig()
			s = serial.New(config.Reload.Com)
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
	app.Version = "v0.1.1"
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
			fmt.Println("start watch")
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
						send(fmt.Sprintf("cd %s", config.Reload.Dir))
						send(fmt.Sprintf("./bm set -s disable"))
						send(fmt.Sprintf("./bm set -d enable"))
						time.Sleep(time.Second * 1)
						send(fmt.Sprintf("./bm install -p %s", hap))
						// start app
						fmt.Println("start...")
						time.Sleep(time.Second * 1)
						send(fmt.Sprintf("./aa start -p %s -n %s", config.Reload.BundleName, config.Reload.AbilityName))
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
			reader := bufio.NewReader(os.Stdin)
			fmt.Println("input quit to exit")
			for {
				line, _, _ := reader.ReadLine()
				msg := string(line)
				if msg == "quit" {
					os.Exit(0)
				}
				sendAndRead(msg)
			}
			return nil
		},
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
			lhap := path.Join(config.Build.Nfsdir, hap)
			_, err := utils.Copy(form, lhap)
			if err != nil {
				return err
			}
			fmt.Println("install...")
			send(fmt.Sprintf("cd %s", config.Reload.Dir))
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
			cmd := fmt.Sprintf("cd %s", config.Reload.Dir)
			send(cmd)
			send(fmt.Sprintf("./bm uninstall -n %s", c.Args().First()))
			return nil
		},
	}
}
