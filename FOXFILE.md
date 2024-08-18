# Foxfile proposals

There should be a way to create foxbox images.

## Merging box images

In OCI, container images are built in a linear fashion, which is the
smart way to build images when you want to not waste massive amounts of
space. But what if we could merge two images?

```dockerfile
from alpine:latest

merge golang:1.21.0 valkey/valkey:8.0
```

This would create a merged image based on alpine with the Go toolchain
and Valkey. It could be assembled from the layers above the `alpine` base
layer.

Questions for implementation:

1. What if an image is not built with the base image (i.e. `alpine` here)?
1. What if there are conflicting files or changes in the merged layers,
   for example, what if both `golang` and `valkey/valkey` have installed
   different versions of an apk package? If we base all our images
   exlusively on `alpine`, could we isolate this via virtual
   environments?
1. Is this even useful enough?

## Faster dependency installs

I like the idea behind Nix but I’m more familiar with containers. The
main idea behind [merging box images](#merging-box-images) is that we
don’t have to build dependencies that have already been built for our
target platform.

For example, if I want to use Ruby and JavaScript in a single image, and
the version I need is not available from the default package manager, I’d
like to be able to download these toolchains in the image without having
to build or manually copy from an existing image.

```dockerfile
from alpine:latest

install node@1.20.1
```

When a precompiled binary is not available, we could still build it
ourselves or copy from an existing image.
