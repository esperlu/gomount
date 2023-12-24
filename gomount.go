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
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Path to config file, mount info file and version number
const (
	confFile      = "/home/jeanluc/.config/gomount/gomount.yaml"
	mountInfoFile = "/proc/self/mountinfo"
	ver           = "v1.2"
)

type Server struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Config struct {
	Servers []Server `yaml:"Servers"`
}

// https://zetcode.com/golang/terminal-colour/
// https://stackoverflow.com/questions/4842424/list-of-ansi-color-escape-sequences
const (
	cReset = "\033[0m"
	cRed   = "\033[31m"
	cGreen = "\033[32m"
)

// Initialise and parse flags
var flagVerbosity = flag.Bool("v", false, "Increased verbosity.")
var flagTimeout = flag.Int("t", 150, "Change default timeout for ping (150 ms).")

func main() {

	flag.Parse()

	var startTime = time.Now()

	// var to store the unmarshalled config's
	var config Config

	// Read yaml config file
	yamlFile, err := os.ReadFile(confFile)
	if err != nil {
		log.Fatalf("yamlFile.Get err   #%v ", err)
	}
	// Parse config file
	err = yaml.Unmarshal(yamlFile, &config)

	// Get list of already mounted hosts
	m, err := os.ReadFile(mountInfoFile)
	if err != nil {
		log.Fatalf("\n Could not find: %s\n Check the path and file name in the const block.\n\n", mountInfoFile)
	}

	mountInfo := string(m)

	// mount servers in goroutines
	var wg sync.WaitGroup
	ch := make(chan string, len(config.Servers))

	for i := range config.Servers {
		server := &config.Servers[i]

		wg.Add(1)
		go func(server Server) {
			defer wg.Done()

			outputString := ""
			// already mounted in /proc/self/mountinfo --> exit goroutine
			if strings.Contains(mountInfo, server.Path) {
				outputString += fmt.Sprintf("%-20s %-15s\n", server.Name, "already mounted")
				ch <- outputString
				return
			}

			// host is not responding on TCP probe --> exit goroutine
			err := goping("tcp", &server.Host, &server.Port, time.Duration(*flagTimeout))
			if err != nil {
				errMsg := "not responding"
				if *flagVerbosity {
					errMsg = err.Error()
				}
				outputString += fmt.Sprintf("%s%-20s %-16s%s\n", cRed, server.Name, errMsg, cReset)
				ch <- outputString
				return
			}

			// execute the mount(8)
			cmd := exec.Command("mount", server.Path)
			output, err := cmd.CombinedOutput()
			if err != nil {
				errMsg := "mount error (increase verbosity with option -v)"
				if *flagVerbosity {
					errMsg = strings.TrimRight(string(output), "\n")
				}
				outputString += fmt.Sprintf("%s%-20s %-16s%s\n", cRed, server.Name, errMsg, cReset)
				ch <- outputString
				return
			}
			outputString += fmt.Sprintf("%-20s mounted\n", server.Name)
			ch <- outputString
		}(*server)
	}
	wg.Wait()
	close(ch)

	// Read channel
	var values []string
	for val := range ch {
		values = append(values, val)
	}

	// Sort the output to preserve the arguments order
	sort.Strings(values)

	// Print final output (omitting the int used to sort the args)
	fmt.Print("\n")
	for _, value := range values {
		fmt.Print(value)
	}

	// print timing
	fmt.Printf(
		"\n%s %s compiled with %s | %.3f sec.\n\n",
		filename(os.Args[0]),
		ver,
		runtime.Version(),
		time.Since(startTime).Seconds(),
	)
}

// goping http ping to check if a server is up
func goping(protocole string, host *string, port *string, t time.Duration) error {
	t = time.Duration(t * time.Millisecond)
	_, err := net.DialTimeout(protocole, *host+":"+*port, t)
	return err
}

func filename(path string) string {
	pos := strings.LastIndex(path, "/")
	return path[pos+1:]
}
