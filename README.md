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

## Usage

For now, clone the repo and then run `go run ./cmd/foxbox` to access the
cli tool.

To run a foxbox, you need a compatible image. So far, I’ve only used the
[alpine 3.18.4 rootfs][alpine]. Download that rootfs to `./images` The
resulting file name excluding the file extention (must be `.tar` or
`.tar.gz`), e.g. `alpine-minirootfs-3.18.4-x86_64` is the image name in
foxbox.

To create a foxbox and run a shell, execute the following in the project
root:

```sh
go run ./cmd/foxbox run --rm alpine-3.18.4-x86_64
```

Run the `hostname` to get the box name. To find the rootfs, head to
`./runtime/BOXNAME/boxfs` on the host machine, where `BOXNAME` is the
hostname of the box.

[alpine]: https://dl-cdn.alpinelinux.org/alpine/v3.18/releases/x86_64/alpine-minirootfs-3.18.4-x86_64.tar.gz

## Features

While interop is really cool, I’m not planning to make foxbox compatible
with the OCI runtime spec, though it might be a fun challenge later on.

- [x] Create and delete foxboxes from rootfs tarballs
- [x] List all foxboxes
- [ ] List running foxes
- [ ] Enter foxboxes by with [nsenter][nsenter]
- [ ] Box inspect (analog to `podman inspect`)
- [ ] Run foxes detached
- [ ] Store logs
- Isolation
  - [x] User namespaces
  - [x] Dropping kernel capabilities
  - [x] Syscall restriction with seccomp
  - [x] Standard streams (stdin, stdout, stderr)
  - [x] `/dev/{null,zero,urandom,random,tty}` access
  - [x] Cgroups v2 (cpu, memory, pids)
- Networking
  - [ ] Host networking
  - [x] slirp4netns
  - [ ] Port forwarding
- Image management
  - [ ] List/show images
  - [ ] Pull images from registry
  - [ ] Remove images
  - [ ] Building images (Boxfile? Foxfile?)
- Volumes
  - [ ] Global volumes
  - [ ] Local volumes (`-v $(pwd):/workdir`)
  - [ ] tempfs mounts
- [ ] Store foxboxes in a fixed place (e.g. `/var` or `~/.foxbox`)
- [ ] Run as daemon to simplify configs

[ociif]: https://github.com/opencontainers/image-spec
[nsenter]: https://github.com/opencontainers/runc/blob/main/libcontainer/nsenter/README.md

### Nice-to-haves

- [ ] (maybe) Use [OCI Image Format][ociif] for images
