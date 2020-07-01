# gomount
Fast user mount of multiple remote mount points, skipping the responding servers not responding to http probe (ping-ish). Mount points need to be configured in `fstab`. Speed is achieved by using Go routines to concurrently send http probes to the targeted servers and mount the servers. It's fast compared to sequential mounts in a bash script due to the way `mount(8)` tries to connect to remote servers that are down. It takes a couple of seconds before `mount(8)` exits with a `connection reset by peer` error message. This Go program first probes the servers with a short timeout (150 ms) which is usually enough to determine if a server is up and then proceed with the actual mount. Each server is processed in concurrent Go routines.

In a comparison test, this go program is about 20 times faster for a list of 9 mounts on "up" servers and 3 "downs".

This go program executes  `exec.Command(mount(8))` rather than the `mount(2)` used by the Go function `syscall.Mount()` because `mount(2)` requires root privileges.

Linux only. Could possibly also work on OS-X. Not tested.

## config file
You first need to make a config file and set its full access path/name into the constant `confFile` around line 32 of the code.

It consists of 4 fields, comma separated lines with the following pattern:
```
[short name],[mount point as defined in fstab],[host name or IP address],[port to ping]
```
Commented out lines will be ignored and config file will be tested for validity at run time.

### config file example
```
arsule-jeanluc,/home/jeanluc/my-mnt/arsule_jeanluc,arsule,221
# This is a comment
arsule-web,/home/jeanluc/my-mnt/arsule_web,arsule,221
rpi-2,/home/jeanluc/my-mnt/rpi_2,rpi_2,22
ubnt,/home/jeanluc/my-mnt/ubnt,192.168.0.201,2211

```

## Usage
`$ gomount [options]`

### Options
* `-t` Change default timeout for ping (150 ms).
* `-v` Increase verbosity by showing error messages.
* `-h` This help screen.

## Typical output
```
$ gomount

bizzard-jeanluc      already mounted
vps                  mounted
bizzard-web          already mounted
ubnt                 already mounted
vps-jeanluc          mounted
gaubert-smb          not responding  
arsule-jeanluc       not responding  
rpi-2                not responding  
gaubert-sshfs        not responding  
arsule-web           not responding  
vps-web              mounted
```

## Typical output with increased verbosity

```

$ gomount -v

gaubert-sshfs        already mounted
ubnt                 already mounted
gaubert-smb          mount: /home/jeanluc/my-mnt/gaubert_sambaz: No such file or directory
arsule-web           dial tcp 192.168.0.153:221: i/o timeout
arsule-jeanluc       dial tcp 192.168.0.153:221: i/o timeout
rpi-2                dial tcp 192.168.0.156:22: i/o timeout
```
----
(c) Jean-Luc Lacroix