// Package gomount mounts remote servers on local mount points defined in `fstab`
// Mount points are read from a user config file. Run `$ gomount -h` for more details.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
)

// Path to config file, mount info file and version number
const (
	confFile      = "/home/jeanluc/.config/gomount/gomount.yaml"
	mountInfoFile = "/proc/self/mountinfo"
	ver           = "v1.2"
)

// struct to store values of a single server
type Srv struct {
	Host string
	Name string
	Path string
	Port string
}

// struct to store server data from config file
type Config struct {
	Server []struct {
		Host string `yaml:"host"`
		Name string `yaml:"name"`
		Path string `yaml:"path"`
		Port string `yaml:"port"`
	} `yaml:"Servers"`
}

// https://zetcode.com/golang/terminal-colour/
// https://stackoverflow.com/questions/4842424/list-of-ansi-color-escape-sequences
const (
	cReset = "\033[0m"
	cRed   = "\033[31m"
	// cRed = "\033[38;2;255;0;0m"
	cGreen = "\033[32m"
	// cGreen = "\033[38;2;0;255;0m"
)

// Initialise and parse flags
var flagVerbosity = flag.Bool("v", false, "Increased verbosity.")
var flagTimeout = flag.Int("t", 150, "Change default timeout for ping (150 ms).")

func main() {

	flag.Parse()

	startTime := time.Now()

	// var to store the unmarshalled config's
	var configs Config

	// Read yaml config file
	yamlFile, err := os.ReadFile(confFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	// Parse config file
	err = yaml.Unmarshal(yamlFile, &configs)

	// Get list of already mounted hosts
	m, err := os.ReadFile(mountInfoFile)
	if err != nil {
		log.Fatalf("\n Could not find: %s\n Check the path and file name in the const block.\n\n", mountInfoFile)
	}
	mountInfo := string(m)

	// mount servers in goroutines
	var wg sync.WaitGroup

	for i := range configs.Server {
		server := configs.Server[i]
		srv := Srv{server.Host, server.Name, server.Path, server.Port}

		wg.Add(1)
		go func(srv Srv) {
			defer wg.Done()

			// already mounted in /proc/self/mountinfo --> exit goroutine
			if strings.Contains(mountInfo, srv.Path) {
				fmt.Printf("%-20s %-15s\n", srv.Name, "already mounted")
				return
			}

			// host is not responding on TCP probe --> exit goroutine
			err := goping("tcp", srv.Host, srv.Port, time.Duration(*flagTimeout))
			if err != nil {
				errMsg := "not responding"
				if *flagVerbosity {
					errMsg = err.Error()
				}
				fmt.Printf("%s%-20s %-16s%s\n", cRed, srv.Name, errMsg, cReset)
				return
			}

			// execute the mount(8)
			cmd := exec.Command("mount", srv.Path)
			output, err := cmd.CombinedOutput()
			if err != nil {
				errMsg := "mount error (increase verbosity with option -v)"
				if *flagVerbosity {
					errMsg = strings.TrimRight(string(output), "\n")
				}
				fmt.Printf("%s%-20s %-16s%s\n", cRed, srv.Name, errMsg, cReset)
				return
			}
			fmt.Printf("%-20s mounted\n", srv.Name)

		}(srv)
	}
	wg.Wait()

	// print timing
	programName := filename(os.Args[0])
	fmt.Printf("\n%s %s | %.3f sec.\n\n", programName, ver, time.Since(startTime).Seconds())
}

// goping http ping to check if a server is up
func goping(protocole string, host string, port string, t time.Duration) error {
	t = time.Duration(t * time.Millisecond)
	_, err := net.DialTimeout(protocole, host+":"+port, t)
	return err
}

func filename(path string) string {
	pos := strings.LastIndex(path, "/")
	return path[pos+1:]
}
