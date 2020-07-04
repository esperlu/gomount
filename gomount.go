// Package gomount mounts remote servers on local mount points defined in `fstab`
// Mount points are read from a user config file. Run `$ gomount -h` for more details.
//
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

// Path to config file, mount info file and version number
const (
	confFile      = "/home/jeanluc/.config/gomount/gomount.conf"
	mountInfoFile = "/proc/self/mountinfo"
	ver           = "v1.1"
)

// Flags
var flagVerbosity = flag.Bool("v", false, "Increased verbosity by showing errors.")
var flagTimeout = flag.Int("t", 150, "Change default timeout for ping (150 ms).")

// Parse flags
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

	// open, read and validate config file. If err, show error and terminate main()
	// if no errors, store all config fields in a []server struct
	hosts, err := readConfig(confFile)
	if err != nil {
		fmt.Printf("\n %s\n\n", err)
		return
	}

	// Get list of already mounted hosts
	m, err := ioutil.ReadFile(mountInfoFile)
	if err != nil {
		fmt.Printf("\n Could not find: %s\n Check the path and file name in the const block.\n\n", mountInfoFile)
		return
	}
	mountInfo := string(m)

	// launch processes in goroutines
	fmt.Println()
	var wg sync.WaitGroup
	for _, srv := range hosts {
		wg.Add(1)
		go func(srv server) {
			defer wg.Done()

			// if already mounted in /proc/self/mountinfo --> exit goroutine
			if strings.Contains(mountInfo, srv.Mnt) {
				fmt.Printf("%-20s %-15s\n", srv.Name, "already mounted")
				return
			}

			// if no port is given in config file --> exit goroutine
			if srv.Port == "" {
				fmt.Printf("%-20s no port given\n", srv.Name)
				return
			}

			// if host is not responding on TCP probe (netcat in go) --> exit goroutine
			err := goping("tcp", srv.Host, srv.Port, time.Duration(*flagTimeout))
			if err != nil {
				errMsg := "not responding"
				if *flagVerbosity {
					errMsg = err.Error()
				}
				fmt.Printf("%-20s %-16s\n", srv.Name, errMsg)
				return
			}

			// execute the mount(8)
			cmd := exec.Command("mount", srv.Mnt)
			output, err := cmd.CombinedOutput()
			if err != nil {
				errMsg := "mount error (increase verbosity with option -v)"
				if *flagVerbosity {
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

// Functions

// goping http ping to check if a server is up
func goping(protocole string, host string, port string, t time.Duration) error {
	t = time.Duration(t * time.Millisecond)
	_, err := net.DialTimeout(protocole, host+":"+port, t)
	return err
}

// checkeErr check err and log.Fatal if any
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// readConfig validates and process the config file.
func readConfig(confFile string) ([]server, error) {
	var hosts []server

	// Open and read config file
	f, err := os.Open(confFile)
	if err != nil {
		return hosts, fmt.Errorf("Could not find the config file: %s\n Check the path and file name in the const block", confFile)
	}
	defer f.Close()

	// Scan through config file
	var lineNumber int
	var sErr []error
	var numberedLines string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNumber++
		// read line from scanner
		l := strings.TrimSpace(scanner.Text())
		numberedLines += fmt.Sprintf("%3d: %s\n", lineNumber, l)

		// skip empty and commented out lines
		if strings.HasPrefix(l, "#") || l == "" {
			continue
		}

		// parse line
		fields := strings.Split(l, ",")

		// Missing fields
		if len(fields) < 4 {
			sErr = append(sErr, fmt.Errorf("%3d: field(s) missing expecting 4 fields, seen %d", lineNumber, len(fields)))
			continue
		}
		// empty fields
		if fields[0] == "" || fields[1] == "" || fields[2] == "" || fields[3] == "" {
			sErr = append(sErr, fmt.Errorf("%3d: field(s) missing or empty. Need 4 fields", lineNumber))
			continue
		}
		// mount point not a dir
		if _, err := os.Stat(fields[1]); os.IsNotExist(err) {
			sErr = append(sErr, fmt.Errorf("%3d: \"%s\" mount point is not a dir", lineNumber, fields[1]))
			continue
		}
		// port to ping not int number
		if _, err := strconv.Atoi(fields[3]); err != nil {
			sErr = append(sErr, fmt.Errorf("%3d: port number empty or not valid: \"%s\" not numerical", lineNumber, fields[3]))
			continue
		}

		// No error for this line. Store fields in a []server struct and loop for next line
		if len(sErr) == 0 {
			hosts = append(hosts, server{
				Name: fields[0],
				Mnt:  fields[1],
				Host: fields[2],
				Port: fields[3],
			})
		}

	}

	// if errors were seen in above loop, print them along with config file if verbosity option -v (verbosity) is set
	if len(sErr) > 0 {
		if *flagVerbosity {
			var e string
			for _, v := range sErr {
				e += fmt.Sprintf("%v\n", v)
			}
			return hosts, fmt.Errorf("\n%s\nErrors:\n%s", numberedLines, e)

		}
		return hosts, fmt.Errorf("error reading config file, aborting. Use option -v to show error(s)")
	}

	if len(hosts) == 0 {
		return hosts, fmt.Errorf("No hosts found in config file. Nothing to mount")
	}

	return hosts, nil
}
