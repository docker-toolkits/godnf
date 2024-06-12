I have written a minimal package manager using Go, which currently only has the install functionality, 
and plan to continuously add features like remove and query in the future. 

## motivation 
in container environments, using RPM package managers results in too many dependencies, 
and the installation size is at least 100+ MB, which significantly affects our usage. 
In the context of container image building, a package manager is indispensable. 
Trimming down the existing dnf (C) functionality is very complex. Therefore, 
I ultimately chose to write a simple package manager using Go.

Like Redhat:
| name         | size   | Desc                     |
|--------------|--------|--------------------------|
| ubi9-minimal | 98.7MB | microdnf package manager |
| ubi9         | 230M   | dnf package manager      |

But godnf just 4.5M. We just add /etc/os-release, repo file, static busybox, then it already meets many scenarios.
| name  | size | Desc                   |
|-------|------|------------------------|
| godnf | 6.8M | godnf packager manager |



usage:
```shell
NAME:
   godnf - package manager use go

USAGE:
   godnf [global options] command [command options] [arguments...]

VERSION:
   v1.0.0

COMMANDS:
   install  install rpm packages
   list     list rpm packages
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --loglevel value  set log level: 0-DEBUG, 1-INFO, 2-WARN, 3-ERROR, default:3 (default: 3)
   --help, -h        show help
   --version, -v     print the version
```

install:
```
godnf install python dnf
```

list:
```shell
godnf list
godnf list --installed
```
