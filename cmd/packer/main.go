package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	"github.com/chop-dbhi/data-models-packer"

	"log"
	"os"
	"text/template"
)

func main() {
	var (
		cfg    = new(packer.Config)
		out    string
		inputs []string
		err    error
	)

	flag.StringVar(&cfg.Comp, "comp", "", "The compression method to be used: '.zip', '.tar.gz', '.tar.gzip', '.tar.bz2', or '.tar.bzip2'. If omitted, the '.tar.gz' method will be used for packing and the file extension will be used to infer a method for unpacking or the STDIN stream is assumed to be uncompressed.")
	flag.StringVar(&cfg.DataVersion, "dataVersion", "", "The specific version of the data in the package.")
	flag.StringVar(&cfg.Etl, "etl", "", "The URL of the ETL code used to generate data. Should be specific to the version of code used and remain that way over time.")
	flag.StringVar(&cfg.KeyPassPath, "keyPassPath", "", "The filepath to the file containing the passphrase needed to access the private key. If omitted, the 'PACKER_KEYPASS' environment variable will be used, if that is unset, the private key is assumed to be unprotected.")
	flag.StringVar(&cfg.KeyPath, "keyPath", "", "The filepath to the public key to use for encrypting packaged data or to the private key to use for unpacking encrypted data. If omitted, the data is assumed to be unencrypted.")
	flag.StringVar(&cfg.Model, "model", "", "The data model to operate against.")
	flag.StringVar(&cfg.ModelVersion, "modelVersion", "", "The specific version of the model to operate against. Defaults to the latest version of the model.")
	flag.StringVar(&out, "out", "", "The directory or filename that should be written to. If omitted, data will be unpacked into the current directory or packed onto STDOUT.")
	flag.StringVar(&cfg.Service, "service", "", "The URL of the data models service to use for fetching schema information.")
	flag.StringVar(&cfg.Site, "site", "", "The site that generated the data.")
	flag.BoolVar(&cfg.VerifyOnly, "verifyOnly", false, "Only verify an existing 'metadata.csv' file in the given data directory. Do not package the directory.")

	flag.Parse()
	inputs = flag.Args()

	switch len(inputs) {

	case 0:

		// Input is from STDIN. Unpack it.
		var packageReader *packer.PackageReader

		// Update config with proper command line arguments.
		cfg.DataDirPath = out
		cfg.PackagePath = ""
		if err = cfg.Verify(); err != nil {
			log.Fatalf("packer: configuration error: %s", err)
		}

		if packageReader, err = packer.NewPackageReader(cfg); err != nil {
			log.Fatalf("packer: error creating unpacking writer: %s", err)
		}

		defer packageReader.Close()

		// Write unpacked package.
		if err = packageReader.Unpack(); err != nil {
			log.Fatalf("packer: error writing unpacked package: %s", err)
		}

		// Verify the metadata file.
		cfg.VerifyOnly = true

		if err = packer.CreateOrVerifyMetadataFile(cfg); err != nil {
			log.Fatalf("packer: error verifying metadata file: %s", err)
		}

	case 1:

		var inputIsDir bool

		// Determine if input path is a directory.
		if inputIsDir, err = packer.IsDir(inputs[0]); err != nil {
			log.Fatalf("packer: error inspecting input path: %s", err)
		}

		// Input path is a directory. Pack it.
		if inputIsDir {

			var packageWriter *packer.PackageWriter

			// Update config with proper command line arguments.
			cfg.DataDirPath = inputs[0]
			cfg.PackagePath = out
			if err = cfg.Verify(); err != nil {
				log.Fatalf("packer: configuration error: %s", err)
			}

			// Create or verify the metadata file.
			if err = packer.CreateOrVerifyMetadataFile(cfg); err != nil {
				log.Fatalf("packer: error creating or verifying metadata file: %s", err)
			}

			// Exit if only metadata verification requested.
			if cfg.VerifyOnly {
				return
			}

			// Create package writer.
			if packageWriter, err = packer.NewPackageWriter(cfg); err != nil {
				log.Fatalf("packer: error creating package writer: %s", err)
			}

			defer packageWriter.Close()

			// Write package to output.
			if err = packageWriter.Pack(); err != nil {
				log.Fatalf("packer: error writing package: %s", err)
			}

			return
		}

		// Input path is a file. Unpack it.
		var packageReader *packer.PackageReader

		// Update config with proper command line arguments.
		cfg.DataDirPath = out
		cfg.PackagePath = inputs[0]
		if err = cfg.Verify(); err != nil {
			log.Fatalf("packer: configuration error: %s", err)
		}

		if packageReader, err = packer.NewPackageReader(cfg); err != nil {
			log.Fatalf("packer: error creating unpacking writer: %s", err)
		}

		defer packageReader.Close()

		// Write unpacked package.
		if err = packageReader.Unpack(); err != nil {
			log.Fatalf("packer: error writing unpacked package: %s", err)
		}

		// Verify the metadata file.
		cfg.VerifyOnly = true

		if err = packer.CreateOrVerifyMetadataFile(cfg); err != nil {
			log.Fatalf("packer: error verifying metadata file: %s", err)
		}

	default:
		log.Fatalf("packer: too many inputs (more than one): '%s'", strings.Join(inputs, ", "))
	}
}

var usage = `Data Models Packer {{.Version}}

Usage:

%s
`

var functionality = `
If the final argument is the path to a directory, it will be packed into the specified file or onto STDIN. If it is the path to a file, it will be unpacked.

Examples:

  # Pack a directory into a file.
  data-models-packer -out test.tar.gz.gpg data/test

  # Verify an existing metadata.csv file only.
  data-models-packer -verifyMetadata data/test

  # Unpack an unencrypted package into a directory.
  data-models-packer -out data/test test.tar.gz

  # Unpack an encrypted data archive (with the  passphrase in a file).
  data-models-packer -keyPath key.asc -keyPassPath  pass.txt test.tar.gz.gpg

  # Unpack an encrypted data archive (with the  passphrase in an env var).
  PACKER_KEYPASS=foobar data-models-packer -keyPath  key.asc test.tar.gz.gpg

Source: https://github.com/chop-dbhi/data-models-packer
`

func init() {
	var buf bytes.Buffer

	cxt := map[string]interface{}{
		"Version": packer.Version,
	}

	template.Must(template.New("usage").Parse(usage)).Execute(&buf, cxt)

	usage = buf.String()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, functionality)
	}
}
