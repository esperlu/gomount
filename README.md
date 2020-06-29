# gomount
Fast user mount of multiple remote mount points skipping the responding servers not responding to http probe (ping-ish). Mount points need to be configured in `fstab`. Speed is achieved by using go's concurrency routines to send http probes to the targeted servers.

Linux only. Could possibly also work on OS-X. Not tested.

## config file
It consists of 4 fields, comma separated lines with the fommowing pattern:
```
[short name] [mount point as defined in fstab] [host name or IP address]  [port to ping]
```
Commented out lines will be ignored.

### Example
```
arsule-jeanluc,/home/jeanluc/my-mnt/arsule_jeanluc,arsule,221
# This is a comment
arsule-web,/home/jeanluc/my-mnt/arsule_web,arsule,221
rpi-2,/home/jeanluc/my-mnt/rpi_2,rpi_2,22
ubnt,/home/jeanluc/my-mnt/ubnt,192.168.0.201,2211

```

## Usage
gomount [options]

### Options
* -t Change default timeout for ping (150 ms).
* -v Increased verbosity by showing OS error messages.
* -c Show config file and its location.
* -h This help screen.

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

#### Author: Jean-Luc Lacroix