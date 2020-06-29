package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

// server
type server struct {
	Name string
	Mnt  string
	Host string
	Port string
}

// Path to config file and version number
const (
	confFile = "/home/jeanluc/.config/gomount/gomount.conf"
	ver      = "v1.0"
)

// Flag debug
var flagDebug = flag.Bool("v", false, "Increased verbosity by showing OS error messages.")
var flagTimeout = flag.Int("t", 150, "Change default timeout for ping (150 ms).")
var flagShowConf = flag.Bool("c", false, "Show config file and its location")

// init flags
func init() {
	flag.Usage = func() {
		fmt.Printf("\nUSAGE:\n  %s [OPTIONS]\n\nOPTIONS:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Printf("  -h    This help\n\n")
	}
	flag.Parse()
}

func main() {

	startTime := time.Now()

	// Open and read config file
	f, err := os.Open(confFile)
	if err != nil {
		fmt.Printf(
			"\nCould not find the config file: %s\nChange the value of the \"confile\" const in code\n\n",
			confFile,
		)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var host []server
	var lineNumber int

	if *flagShowConf {
		fmt.Printf("\n%s\n\n", confFile)
	}

	for scanner.Scan() {
		lineNumber++

		// Show config file
		if *flagShowConf {
			fmt.Println(scanner.Text())
			continue
		}

		// Skip commented lines
		confLine := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(confLine, "#") {
			continue
		}
		// Validate config file
		fields := strings.Split(confLine, ",")
		if len(fields) < 4 {
			fmt.Printf("\nERROR: Field missing in config file on line %d:\n --> %s\nExpecting 4 fields, seen %d.\n\n", lineNumber, confLine, len(fields))
			continue
		}
		if _, err := strconv.Atoi(fields[3]); err != nil {
			fmt.Printf("\nERROR: Port number empty or not INT on line %d:\n --> %s\n\"%s\" not numerical.\n\n", lineNumber, confLine, fields[3])
			continue
		}

		//  Store config file lines into []struc
		host = append(host, server{
			Name: fields[0],
			Mnt:  fields[1],
			Host: fields[2],
			Port: fields[3],
		})
		checkErr(err)
	}

	// Get list of already mounted hosts
	data, _ := ioutil.ReadFile("/proc/self/mountinfo")
	mountInfo := string(data)

	fmt.Println()
	var wg sync.WaitGroup
	for _, srv := range host {

		// launch processes in goroutines in a closure func
		wg.Add(1)
		go func(srv server) {
			defer wg.Done()
			// check if already mounted in /proc/self/mountinfo
			if strings.Contains(mountInfo, srv.Mnt) {
				fmt.Printf("%-20s %-15s\n", srv.Name, "already mounted")
				return
			}

			// check if port is given
			if srv.Port == "" {
				fmt.Printf("%-20s no port given\n", srv.Name)
				return
			}

			// check if host is responding on TCP probe (netcat in go)
			err := goping("tcp", srv.Host, srv.Port, time.Duration(*flagTimeout))
			if err != nil {
				errMsg := "not responding"
				if *flagDebug {
					errMsg = err.Error()
				}
				fmt.Printf("%-20s %-16s\n", srv.Name, errMsg)
				return
			}

			cmd := exec.Command("mount", srv.Mnt)
			output, err := cmd.CombinedOutput()
			if err != nil {
				errMsg := "mount error (try option -v verbose)"
				if *flagDebug {
					// remove \n from output
					errMsg = string(output[:len(output)-1])
				}
				fmt.Printf("%-20s %-16s\n", srv.Name, errMsg)
			} else {
				fmt.Printf("%-20s mounted\n", srv.Name)
			}
		}(srv)

	}
	// Wait for all the routines to finish
	wg.Wait()

	fmt.Printf("\n%s %s | %.3f sec.\n\n", path.Base(os.Args[0]), ver, time.Since(startTime).Seconds())
}

// goping
func goping(protocole string, host string, port string, t time.Duration) error {
	t = time.Duration(t * time.Millisecond)
	_, err := net.DialTimeout(protocole, host+":"+port, t)
	return err
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
