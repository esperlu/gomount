# gomount
Fast user mount of multiple remote mount points, skipping the responding servers not responding to http probe (ping-ish). Mount points need to be configured in `fstab`. Speed is achieved by using Go routines to concurrently send http probes to the targeted servers and mount the servers. It's fast compared to sequential mounts in a bash script due to the way `mount(8)` tries to connect to remote servers that are down. It takes a couple of seconds before `mount(8)` exits with a `connection reset by peer` error message. This Go program first probes the servers with a short timeout (150 ms) which is usually enough to determine if a server is up and then proceed with the actual mount. Each server is processed in concurrent Go routines.

In a comparison test, this Go program is about 20 times faster for a list of 9 mounts on "up" servers and 3 "downs".

This go program executes  `exec.Command(mount(8))` rather than the `mount(2)` used by the Go function `syscall.Mount()` because `mount(2)` requires root privileges.

Linux only. Could possibly also work on OS-X. Not tested.

## config file
You first need to make a YAML config file and set its full access path/name into the constant `confFile` in the Go code.


### config file example

    Servers:

    - name: rpi-2
      path: /home/jeanluc/my-mnt/rpi_2
      host: 192.168.10.55
      port: 22

    - name: arsule-jeanluc
      path: /home/jeanluc/my-mnt/arsule_jeanluc
      host: arsule
      port: 22

    - name: arsule-web
      path: /home/jeanluc/my-mnt/arsule_web
      host: arsule
      port: 22


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