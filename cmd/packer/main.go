package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/chop-dbhi/data-models-packer"

	"log"
	"os"
	"text/template"
)

func main() {
	var (
		inputComp    string
		dataVersion  string
		etl          string
		keyPassPath  string
		keyPath      string
		model        string
		modelVersion string
		out          string
		service      string
		site         string
		verifyOnly   bool
		inputs       []string
	)

	flag.StringVar(&inputComp, "comp", "", "The compression method to be used: '.zip', '.tar.gz', '.tar.gzip', '.tar.bz2', or '.tar.bzip2'. If omitted, the '.tar.gz' method will be used for packing and the file extension will be used to infer a method for unpacking or the STDIN stream is assumed to be uncompressed.")
	flag.StringVar(&dataVersion, "dataVersion", "", "The specific version of the data in the package.")
	flag.StringVar(&etl, "etl", "", "The URL of the ETL code used to generate data. Should be valid and accurate over time.")
	flag.StringVar(&keyPassPath, "keyPassPath", "", "The filepath to the file containing the passphrase needed to access the private key. If omitted, the 'PACKER_PRIPASS' environment variable will be used, if that is unset, the private key is assumed to be unprotected.")
	flag.StringVar(&keyPath, "keyPath", "", "The filepath to the public key to use for encrypting packaged data or to the private key to use for unpacking encrypted data. If omitted, the data is assumed to be unencrypted.")
	flag.StringVar(&model, "model", "", "The data model to operate against.")
	flag.StringVar(&modelVersion, "modelVersion", "", "The specific version of the model to operate against. Defaults to the latest version of the model.")
	flag.StringVar(&out, "out", "", "The directory or filename that should be written to. If omitted, data will be unpacked into the current directory or packed onto STDOUT.")
	flag.StringVar(&service, "service", "", "The URL of the data models service to use for fetching schema information.")
	flag.StringVar(&site, "site", "", "The site that generated the data.")
	flag.BoolVar(&verifyOnly, "verifyOnly", false, "Only verify an existing 'metadata.csv' file in the given data directory. Do not package the directory.")

	flag.Parse()
	inputs = flag.Args()

	switch lenInputs := len(inputs); {
	case lenInputs > 1:
		log.Fatalf("packer: too many inputs (more than one): '%s'", strings.Join(inputs, ", "))

	case lenInputs == 1:

		var (
			err           error
			inputFile     *os.File
			inputFileInfo os.FileInfo
		)

		// Open input path.
		if inputFile, err = os.Open(inputs[0]); err != nil {
			log.Fatalf("packer: error opening input path: %s", err)
		}

		defer inputFile.Close()

		// Stat input path to find out if it is a file or a directory.
		if inputFileInfo, err = inputFile.Stat(); err != nil {
			log.Fatalf("packer: error stat-ing input: %s", err)
		}

		// Input path is a directory. Pack it.
		if inputFileInfo.IsDir() {

			var (
				writer     io.Writer
				outCloser  io.Closer
				encCloser  io.Closer
				compCloser io.Closer
				compWriter *packer.CompressingWriter
				filePacker filepath.WalkFunc
			)

			// Create or verify the metadata file.
			if err = packer.CreateOrVerifyMetadataFile(inputs[0], site, model, modelVersion, etl, dataVersion, service, verifyOnly); err != nil {
				log.Fatalf("packer: error creating or verifying metadata file: %s", err)
			}

			if verifyOnly {
				return
			}

			// Create output writer.
			if outCloser, err = createOutputWriter(&writer, out); err != nil {
				log.Fatalf("packer: error opening package output for writing: %s", err)
			}

			defer outCloser.Close()

			// Add encryption to the writer.
			if inputComp != ".zip" {

				if encCloser, err = addEncryption(&writer, keyPath); err != nil {
					log.Fatalf("packer: error adding encryption to package output: %s", err)
				}

				defer encCloser.Close()

			} else {
				// Encrypted zip files not supported on unpack.
				log.Print("packer: encryption not supported with zip compression, outputting unencrypted package")
			}

			// Add compression to the writer.
			if compCloser, err = addCompression(&writer, inputComp); err != nil {
				log.Fatalf("packer: error adding compression to package output: %s", err)
			}

			defer compCloser.Close()

			// Type assert writer to a CompressingWriter.
			compWriter = writer.(*packer.CompressingWriter)

			// Make a filepath.WalkFunc to pack files into the package.
			filePacker = makeFilePacker(inputs[0], compWriter)

			// Write the files into a package.
			if err = filepath.Walk(inputs[0], filePacker); err != nil {
				log.Fatalf("packer: error writing package: %s", err)
			}

			return
		}

		// Input path is a file. Unpack it.
		var (
			reader       io.Reader
			encrypted    bool
			comp         string
			decompReader *packer.DecompressingReader
		)

		// Create a basic reader from the input file.
		reader = io.Reader(inputFile)

		// Get filename extensions.
		if encrypted, comp, err = handleExtensions(inputFileInfo.Name(), inputComp); err != nil {
			log.Fatalf("packer: error collecting file extensions: %s", err)
		}

		// Add decryption to the reader if necessary.
		if encrypted {

			if err = addDecryption(&reader, keyPath, keyPassPath); err != nil {
				log.Fatalf("packer: error adding decryption: %s", err)
			}
		}

		// Add decompression to the reader.
		if reader, err = packer.NewDecompressingReader(reader, comp, inputFileInfo.Size()); err != nil {
			log.Fatalf("packer: error adding decompression: %s", err)
		}

		// Type assert reader to a decompressing reader
		decompReader = reader.(*packer.DecompressingReader)

		// If output directory not specified, use current working directory.
		if out == "" {
			if out, err = os.Getwd(); err != nil {
				log.Fatalf("packer: error getting current working directory: %s", err)
			}
		}

		// Write unpacked package.
		if err = writeUnpacked(out, decompReader); err != nil {
			log.Fatalf("packer: error writing unpacked package: %s", err)
		}

		// Verify the metadata file.
		if err = packer.CreateOrVerifyMetadataFile(inputs[0], site, model, modelVersion, etl, dataVersion, service, true); err != nil {
			log.Fatalf("packer: error verifying metadata file: %s", err)
		}

	case lenInputs == 0:
		// TODO: Implement unpacking from STDIN.
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
  PACKER_PRIPASS=foobar data-models-packer -keyPath  key.asc test.tar.gz.gpg

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
