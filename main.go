/*
  Copyright (c) 2018, Sylabs, Inc. All rights reserved.
  This software is licensed under a 3-clause BSD license.  Please
  consult LICENSE file distributed with the sources of this project regarding
  your rights to use or distribute this software.
*/

/*
This is a proof of concept for docker / registry container pulls to
Singularity sandboxes, implemented initially as a clone of the
functionality of docker2singularity

2018-02-16 David Trudgian <dtrudg@sylabs.io>
*/

package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/containers/image/copy"
	"github.com/containers/image/signature"
	"github.com/containers/image/transports"
	"github.com/containers/image/transports/alltransports"
	imgspecv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/opencontainers/image-tools/image"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()

	app.Name = "docker2singularity-go"
	app.Version = "0.0.1"
	app.Usage = "Build a Singularity sandbox from a docker registry"
	app.UsageText = "docker2singularity-go docker_ref sandbox_dir"
	app.Description = fmt.Sprintf(`Pulls a docker container from the <docker_ref> URI, which refers to
a docker registry, and creates a Singularity sandbox container from it.

Supports standard docker registry URIs by prefixing with docker://, e.g.

    docker2singularity-go docker://ubuntu:latest ubuntu_sandbox
    docker2singularity-go docker://quay.io/biocontainers/canu canu_sandbox
    docker2singularity-go docker://localhost:5000/myubuntu myubuntu_sandbox

Full list of transports supported:
%s


Will authenticate to repositories using settings in $HOME/.docker/config.json

`, strings.Join(transports.ListNames(), ", "))
	app.Author = "David C. Trudgian"
	app.Email = "dtrudg@sylabs.io"
	app.Copyright = "(c) 2018 Sylabs Inc."

	app.Action = func(c *cli.Context) error {
		if len(os.Args) != 3 {
			fmt.Println("Usage: " + app.UsageText)
			os.Exit(1)
		}
		CreateSandbox(os.Args[1], os.Args[2])
		return nil
	}

	app.Run(os.Args)

	return
}

// CreateSandbox - Given a uri, with a scheme supported as a containers/image
// transport, pull the image, unpack it to sandBoxDir and setup singularity
// specific environment.
func CreateSandbox(uri string, sandboxDir string) {

	ociDir, err := fetchImage(uri)
	errFatal(err)

	imgConfig, err := getOCIConfig(ociDir)

	defer os.RemoveAll(ociDir)

	err = unpackImage(ociDir, sandboxDir)
	errFatal(err)

	err = insertBaseEnv(sandboxDir)
	errFatal(err)

	err = insertRunScript(sandboxDir, imgConfig)
	errFatal(err)

	err = insertEnv(sandboxDir, imgConfig)
	errFatal(err)
}

func fetchImage(uri string) (string, error) {

	log.Print("Setting singature policy (accept all) \n")

	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return "", err
	}

	log.Printf("Fetching container %s \n", uri)

	srcRef, err := alltransports.ParseImageName(uri)
	if err != nil {
		return "", err
	}

	containerDir, err := ioutil.TempDir("", "docker2singularity_")

	log.Printf("Temporary oci directory: %s \n", containerDir)

	destRef, err := alltransports.ParseImageName("oci:" + containerDir + ":singularity2docker")
	if err != nil {
		return "", err
	}

	err = copy.Image(policyContext, destRef, srcRef, &copy.Options{
		ReportWriter: os.Stderr,
	})
	if err != nil {
		return "", err
	}

	return containerDir, nil
}

func unpackImage(ociDir string, sandboxDir string) error {

	refs := []string{"name=singularity2docker"}
	log.Printf("Unpacking container to %s \n", sandboxDir)
	err := (image.UnpackLayout(ociDir, sandboxDir, "amd64", refs))
	return err
}

func getOCIConfig(ociDir string) (imgspecv1.ImageConfig, error) {

	ref, err := alltransports.ParseImageName("oci:" + ociDir + ":singularity2docker")
	errFatal(err)

	img, err := ref.NewImage(nil)
	errFatal(err)

	defer img.Close()

	imgSpec, err := img.OCIConfig()
	errFatal(err)
	imgConfig := imgSpec.Config

	return imgConfig, err

}

func insertBaseEnv(sandBoxDir string) error {

	f, err := os.Open("environment.tar")
	errFatal(err)

	defer f.Close()

	errFatal(Untar(sandBoxDir, f))

	return nil

}

func insertRunScript(sandBoxDir string, ociConfig imgspecv1.ImageConfig) error {

	f, err := os.Create(sandBoxDir + "/.singularity.d/runscript")
	errFatal(err)

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	errFatal(err)
	_, err = f.WriteString(strings.Join(ociConfig.Entrypoint, " "))
	errFatal(err)
	_, err = f.WriteString(" ")
	errFatal(err)
	_, err = f.WriteString(strings.Join(ociConfig.Cmd, " "))
	errFatal(err)
	_, err = f.WriteString("\n")
	errFatal(err)

	f.Sync()

	err = os.Chmod(sandBoxDir+"/.singularity.d/runscript", 0755)
	errFatal(err)

	return nil

}

func insertEnv(sandBoxDir string, ociConfig imgspecv1.ImageConfig) error {

	f, err := os.Create(sandBoxDir + "/.singularity.d/env/10-docker2singularity.sh")
	errFatal(err)

	defer f.Close()

	_, err = f.WriteString("#!/bin/sh\n")
	errFatal(err)

	for _, element := range ociConfig.Env {
		_, err = f.WriteString("export " + element + "\n")
		errFatal(err)
	}

	f.Sync()

	err = os.Chmod(sandBoxDir+"/.singularity.d/env/10-docker2singularity.sh", 0755)
	errFatal(err)

	return nil
}

func errFatal(err error) {
	if err != nil {
		log.Panicf("FATAL ERROR: %v", err)
	}
}

//
// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func Untar(dst string, r io.Reader) error {

	gzr, err := gzip.NewReader(r)
	defer gzr.Close()
	if err != nil {
		return err
	}

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(dst, header.Name)

		// the following switch could also be done using fi.Mode(), not sure if there
		// a benefit of using one vs. the other.
		// fi := header.FileInfo()

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer f.Close()

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
		}
	}
}
