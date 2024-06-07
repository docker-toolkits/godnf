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


