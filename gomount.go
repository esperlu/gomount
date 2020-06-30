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

	// validate config file. Iff err, terminate main()
	err = validateConf(f)
	if err != nil {
		return
	}

	// read config and mount
	host := readConfig(f)

	// Get list of already mounted hosts
	data, _ := ioutil.ReadFile(mountInfoFile)
	mountInfo := string(data)

	// launch processes in goroutines
	fmt.Println()
	var wg sync.WaitGroup
	for _, srv := range host {
		wg.Add(1)
		go func(srv server) {
			defer wg.Done()

			// check if already mounted in /proc/self/mountinfo
			if strings.Contains(mountInfo, srv.Mnt) {
				fmt.Printf("%-20s %-15s\n", srv.Name, "already mounted")
				return
			}

			// check if port is given in config file
			if srv.Port == "" {
				fmt.Printf("%-20s no port given\n", srv.Name)
				return
			}

			// check if host is responding on TCP probe (netcat in go)
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

// goping
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

// readConfig reads the config file. Returns list of servers to mount and mount points.
func readConfig(f *os.File) []server {
	var host []server

	// reset file seek head at bebinning of file (f *os.File pointer may be used by other func)
	defer f.Seek(0, 0)

	// Scan through config file and process the mounts
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		confLine := strings.TrimSpace(scanner.Text())
		fields := strings.Split(confLine, ",")
		// Skip commented lines
		if strings.HasPrefix(confLine, "#") || confLine == "" {
			continue
		}

		//  Store config file lines into []struc
		host = append(host, server{
			Name: fields[0],
			Mnt:  fields[1],
			Host: fields[2],
			Port: fields[3],
		})
	}
	return host
}

// validateConf validates the config file and print errors
func validateConf(f *os.File) error {

	// reset file seek head at bebinning of file (f *os.File pointer may be used by other func)
	defer f.Seek(0, 0)

	// Scan through config file and print lines
	var lineNumber int
	var sErr []error
	var allLines string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lineNumber++
		// read line from scanner
		confLine := strings.TrimSpace(scanner.Text())
		allLines += fmt.Sprintf("%3d: %s\n", lineNumber, confLine)
		// skip empty and commented out lines
		if strings.HasPrefix(confLine, "#") || confLine == "" {
			continue
		}
		fields := strings.Split(confLine, ",")
		// Missing fields
		if len(fields) < 4 {
			sErr = append(sErr, fmt.Errorf("%3d: field missing expecting 4 fields, seen %d", lineNumber, len(fields)))
			continue
		}
		// empty fields
		if fields[0] == "" || fields[1] == "" || fields[2] == "" || fields[3] == "" {
			sErr = append(sErr, fmt.Errorf("%3d: field missing or empty. Need 4 fields", lineNumber))
			continue
		}
		// mount point not a dir
		if _, err := os.Stat(fields[1]); os.IsNotExist(err) {
			sErr = append(sErr, fmt.Errorf("%3d: \"%s\" mount point is not a dir", lineNumber, fields[1]))
			continue
		}
		// port to ping not int number
		if _, e := strconv.Atoi(fields[3]); e != nil {
			sErr = append(sErr, fmt.Errorf("%3d: port number empty or not valid: \"%s\" not numerical", lineNumber, fields[3]))
			continue
		}

	}

	// if errors seen, print them along with config file if verbosity option -v is set
	if len(sErr) > 0 {
		if *flagVerbosity {
			fmt.Println("")
			fmt.Println(allLines)
			fmt.Println("Errors:")
			for _, e := range sErr {
				fmt.Printf("%v\n", e)
			}
			fmt.Println("")
		} else {
			fmt.Printf("\nError reading config file, aborting. Use option -v to show error(s).\n\n")
		}
		return fmt.Errorf("Conf file not valid")
	}
	return nil
}
