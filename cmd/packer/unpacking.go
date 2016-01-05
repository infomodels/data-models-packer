package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/chop-dbhi/data-models-packer"
)

// handleExtensions takes a file name and the input compression method and
// returns whether decryption is needed and the standardized compression
// method.
func handleExtensions(name string, inputComp string) (encrypted bool, comp string, err error) {

	var (
		exts        map[string]bool
		conflictErr error
		i           int
		j           int
	)

	exts = map[string]bool{
		"gpg": false,
		"tar": false,
		"bz2": false,
		"gz":  false,
		"zip": false,
	}

	conflictErr = errors.New("extensions conflict with passed compression method")
	j = len(name)

	for i = len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			switch name[i+1 : j] {
			case "gpg":
				exts["gpg"] = true
				j = i
			case "tar":
				exts["tar"] = true
				j = i
			case "bzip2", "bz2":
				exts["bz2"] = true
				j = i
			case "gzip", "gz":
				exts["gz"] = true
				j = i
			case "zip":
				exts["zip"] = true
				j = i
			default:
				return false, "", fmt.Errorf("unexpected file extension: %s\n", name[i+1:j])
			}
		}
	}

	switch {
	case exts["gz"] && exts["bz2"] || exts["gz"] && exts["zip"] || exts["bz2"] && exts["zip"]:
		return false, "", errors.New("incompatible compression extensions found")

	case exts["gz"] && !exts["tar"] || exts["bz2"] && !exts["tar"]:
		return false, "", errors.New("cannot decompress 'bz2' or 'gz' compressed archives without 'tar'")

	case exts["tar"] && exts["gz"]:
		if inputComp != "" && inputComp != ".tar.gz" && inputComp != ".tar.gzip" {
			return false, "", conflictErr
		}
		comp = ".tar.gz"

	case exts["tar"] && exts["bz2"]:
		if inputComp != "" && inputComp != ".tar.bz2" && inputComp != ".tar.bzip2" {
			return false, "", conflictErr
		}
		comp = ".tar.bz2"

	case exts["zip"]:
		if inputComp != "" && inputComp != ".zip" {
			return false, "", conflictErr
		}
		comp = ".zip"

	default:
		return false, "", errors.New("no compression extensions found")
	}

	return exts["gpg"], comp, nil
}

// addDecryption creates a decrypting reader based on the passed reader using
// the files at key path and key pass path if they are passed and may return a
// closer that must be closed.
func addDecryption(reader *io.Reader, keyPath string, keyPassPath string) (err error) {

	var (
		keyReader  *os.File
		passReader io.Reader
		passFile   *os.File
		decReader  io.Reader
	)

	if keyPath == "" {
		return errors.New("no keyfile passed")
	}

	if keyReader, err = os.Open(keyPath); err != nil {
		return err
	}

	defer keyReader.Close()

	passReader = strings.NewReader("")

	if keyPassPath != "" {

		if passFile, err = os.Open(keyPassPath); err != nil {
			return err
		}

		defer passFile.Close()

		passReader = io.Reader(passFile)
	}

	if os.Getenv("PACKER_PRIPASS") != "" {
		passReader = strings.NewReader(os.Getenv("PACKER_PRIPASS"))
	}

	if decReader, err = packer.Decrypt(*reader, keyReader, passReader); err != nil {
		return err
	}

	*reader = decReader
	return nil
}

// writeUnpacked writes files from a package reader to the output directory.
func writeUnpacked(outDir string, r *packer.DecompressingReader) (err error) {

	for {

		var (
			fileHeader *tar.Header
			filePath   string
			fileDir    string
			fileInfo   os.FileInfo
			file       *os.File
		)

		// Advance to next file in the reader or exit with success if there are
		// no more.
		if fileHeader, err = r.Next(); err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		filePath = filepath.Join(outDir, fileHeader.Name)
		fileDir = filepath.Dir(filePath)
		fileInfo = fileHeader.FileInfo()

		// Make directories in file path.
		if err = os.MkdirAll(fileDir, 0766); err != nil {
			return err
		}

		// Open file for writing.
		if file, err = os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fileInfo.Mode()); err != nil {
			return err
		}
		defer file.Close()

		// Write file from the reader.
		log.Printf("packer: unpacking '%s'", filepath.Base(fileHeader.Name))
		if _, err = io.Copy(file, r); err != nil {
			return err
		}
	}
}
