## 🦊📦 foxbox

foxbox is a simplistic, rootless process isolation runtime and CLI tool
based on Linux user namespaces.

It’s a fun project created to explore and better understand what’s behind
the high-level abstractions of existing container runtimes like Docker or
Podman.

A lot of the initial process isolation code is based on [Liz Rice’s
“Containers From Scratch”][lizrice], [Lizzie Dixon’s Linux containers in
500 lines of code”][lizzie500], and runc’s [libcontainer][runc].

Note: better not use foxbox in prod. I run shady fluff in prod and even I
wouldn’t run this there (yet, hehe).

[lizrice]: https://github.com/lizrice/containers-from-scratch
[lizzie500]: https://blog.lizzie.io/linux-containers-in-500-loc.html
[runc]: https://github.com/opencontainers/runc

## Terminology

Purely for fun, containers are named _foxboxes_ (or _boxes_ for short).
Likewise, tasks in containers are just called _foxes_.

## Features

While interop is really cool, I’m not planning to make foxbox compataible
with

- [x] Create and delete foxboxes from rootfs tarballs
- [x] List all foxboxes
- [ ] List running foxes
- [ ] Pull images
- [ ] Enter foxboxes by with [nsenter][nsenter]
- Isolation
  - [x] User namespaces
  - [x] Dropping kernel capabilities
  - [x] Syscall restriction with seccomp
  - [ ] Cgroups v2
  - [ ] `/dev/random` access
- Networking
  - [ ] Host networking
  - [ ] slirp4netns
  - [ ] Port forwarding
- Volumes
  - [ ] Global volumes
  - [ ] Local volumes (`-v $(pwd):/workdir`)
  - [ ] tempfs mounts
- [ ] Store foxboxes in a fixed place (e.g. `/var` or `~/.foxbox`)

[ociif]: https://github.com/opencontainers/image-spec
[nsenter]: https://github.com/opencontainers/runc/blob/main/libcontainer/nsenter/README.md

### Nice-to-haves

- [ ] (maybe) Use [OCI Image Format][ociif] for images
