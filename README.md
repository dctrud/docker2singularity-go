# docker2singularity-go

**PROOF OF CONCEPT - NOT YET TIDY/PRODUCTION CODE!**


This is a reimplementation of docker2singularity that:

  * Is written in Go
  * Is compiled and has no runtime dependency on docker or singularity
  * Creates sandbox containers at this time
  
Has been written as:

  * A proof of concept for creating singularity containers from docker registry
    images in Go
  * A troubleshooting/verification tool for docker container issues 


## Currently Does...

  * Pull from any endpoint supported by containers/image (docker registry, archives, oci layouts etc.) 
  * Build a singularity sandbox with runscript and env set
  
## Does not yet...

  * Properly fix the env yet (need to exclude things, proper quoting etc.)
  * Build an image, only sandbox

## To build

```
$ go get github.com/rancher/trash
$ trash
$ go build
```
If you do not need ostree support, or do not have gpgme available you can use
build tags to exclude those features of `containers/image`:

```
$ go build --tags containers_image_openpgp,containers_image_ostree_stub
```


## Pulling dockerhub images...

You must run where you built it for now!


```bash
01:21 PM $ ./docker2singularity-go docker://ubuntu:latest ubuntu_sandbox 
2018/02/16 13:21:44 Setting singature policy (accept all) 
2018/02/16 13:21:44 Fetching container docker://ubuntu:latest 
2018/02/16 13:21:44 Temporary oci directory: /tmp/docker2singularity_208402548 
Getting image source signatures
Copying blob sha256:1be7f2b886e89a58e59c4e685fcc5905a26ddef3201f290b96f1eff7d778e122
 40.88 MB / 40.88 MB [======================================================] 6s
Copying blob sha256:6fbc4a21b806838b63b774b338c6ad19d696a9e655f50b4e358cc4006c3baa79
 846 B / 846 B [============================================================] 0s
Copying blob sha256:c71a6f8e13782fed125f2247931c3eb20cc0e6428a5d79edb546f1f1405f0e49
 620 B / 620 B [============================================================] 0s
Copying blob sha256:4be3072e5a37392e32f632bb234c0b461ff5675ab7e362afad6359fbd36884af
 854 B / 854 B [============================================================] 0s
Copying blob sha256:06c6d2f5970057aef3aef6da60f0fde280db1c077f0cd88ca33ec1a70a9c7b58
 171 B / 171 B [============================================================] 0s
Copying config sha256:5224c76eca8e908e113254e975054abf205042d9e0ec0c31aba981b28dde9b4b
 2.62 KB / 2.62 KB [========================================================] 0s
Writing manifest to image destination
Storing signatures
2018/02/16 13:21:53 Unpacking container to ubuntu_sandbox 

01:29 PM $ singularity exec ubuntu_sandbox/ cat /etc/lsb-release
DISTRIB_ID=Ubuntu
DISTRIB_RELEASE=16.04
DISTRIB_CODENAME=xenial
DISTRIB_DESCRIPTION="Ubuntu 16.04.3 LTS"
```

## Pulling private docker registry images...

The docker2singularity-go tool can use your `docker login` config to access private
registries, such as the NVIDIA GPU Cloud.

E.g. to pull docker images from the NVIDIA GPU Cloud into a singularity sandbox:

```sh
# First configure your login per the instructions on the GPU cloud site: 
$ docker login nvcr.io
Username: $oauthtoken
Password:
Login Succeeded

# Now you pull a GPU cloud container into a singularity sandbox
$ ./docker2singularity-go docker://nvcr.io/nvidia/caffe:18.01-py2 nvcr_caffe_sandbox/
...

# And run with Singularity...
$ singularity run --nv nvcr_caffe_sandbox/

==================
== NVIDIA Caffe ==
==================

NVIDIA Release 18.01 (build 282448)

Container image Copyright (c) 2017, NVIDIA CORPORATION.  All rights reserved.
Copyright (c) 2014, 2015, The Regents of the University of California (Regents)
All rights reserved.
...
```

## Other formats to Singularity sandboxes...

Through the use of `containers/image` this tool supports more than `docker://`
hub/registry URIs. You can create sandboxes from:

  - docker archives
  - docker-daemon managed containers
  - oci directories
  - oci archives
  - ostree
  - tarballs

