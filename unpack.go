package packer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/bzip2"
	"compress/gzip"

	"errors"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/openpgp"
)

// DecompressingReader wraps zip.Reader and tar.Reader in a consistent API.
type DecompressingReader struct {
	tarReader  *tar.Reader
	gzipReader *gzip.Reader
	zip        bool
	zipReader  *zip.Reader
	zipIndex   int
	zipFile    io.ReadCloser
}

// NewDecompressingReader takes a reader with compressed data, a string
// describing the compression method (".tar.gz", ".tar.bz2", or ".zip"), and
// the size of the reader file and returns a reader that decompresses the data.
func NewDecompressingReader(r io.Reader, compr string, size int64) (reader *DecompressingReader, err error) {

	reader = new(DecompressingReader)

	switch compr {
	case ".zip":

		// Handle zip decompression specially because the interface is
		// different.
		var (
			readerAt io.ReaderAt
			found    bool
		)

		reader.zip = true
		reader.zipIndex = -1

		if readerAt, found = r.(io.ReaderAt); !found {
			return nil, errors.New("failed to assert io.ReaderAt type on reader")
		}

		// BUG(aaron0browne): The zip reader is unable to decompress zip
		// archives that that use the DEFLATE64 compression method.
		if reader.zipReader, err = zip.NewReader(readerAt, size); err != nil {
			return nil, err
		}

	case ".tar.bz2":

		var bzip2Reader io.Reader

		bzip2Reader = bzip2.NewReader(r)
		reader.tarReader = tar.NewReader(bzip2Reader)

	case ".tar.gz":

		// Save the gzipReader for closing later.
		if reader.gzipReader, err = gzip.NewReader(r); err != nil {
			return nil, err
		}

		reader.tarReader = tar.NewReader(reader.gzipReader)
	}

	return reader, nil
}

// Next advances to the next entry in the compressed file.
func (r *DecompressingReader) Next() (header *tar.Header, err error) {

	// Handle underlying zip.Reader specially, since it has different behavior.
	if r.zip {

		var file *zip.File

		r.zipIndex++

		if r.zipIndex < len(r.zipReader.File) {

			file = r.zipReader.File[r.zipIndex]

			if r.zipFile, err = file.Open(); err != nil {
				return nil, err
			}

			if header, err = tar.FileInfoHeader(file.FileHeader.FileInfo(), ""); err != nil {
				return nil, err
			}

			header.Name = file.FileHeader.Name

			return header, nil
		}

		return nil, io.EOF
	}

	return r.tarReader.Next()
}

// Read reads from the current entry in the compressed file.
func (r *DecompressingReader) Read(buf []byte) (n int, err error) {

	if r.zip {

		if n, err = r.zipFile.Read(buf); err == io.EOF {
			r.zipFile.Close()
		}

		return n, err
	}

	return r.tarReader.Read(buf)
}

// Decrypt takes a reader with encrypted data, a string path to the private key
// file, and optionally a string path to the passphrase file and returns a
// reader that decrypts the data. It assumes there is only one OpenPGP entity
// involved.
func Decrypt(encReader io.Reader, keyReader io.Reader, passReader io.Reader) (plainReader io.Reader, err error) {

	var (
		entityList openpgp.EntityList
		entity     *openpgp.Entity
		passphrase []byte
		msgDetails *openpgp.MessageDetails
	)

	// Read armored private key into entityList.
	if entityList, err = openpgp.ReadArmoredKeyRing(keyReader); err != nil {
		return nil, err
	}

	entity = entityList[0]

	// Decode entity private key and subkey private keys. This assumes there is
	// only one entity involved.
	if passphrase, err = ioutil.ReadAll(passReader); err != nil {
		return nil, err
	}

	passphrase = bytes.TrimSpace(passphrase)

	if entity.PrivateKey != nil && entity.PrivateKey.Encrypted {
		if err = entity.PrivateKey.Decrypt(passphrase); err != nil {
			return nil, err
		}
	}

	for _, subkey := range entity.Subkeys {
		if subkey.PrivateKey != nil && subkey.PrivateKey.Encrypted {
			if err = subkey.PrivateKey.Decrypt(passphrase); err != nil {
				return nil, err
			}
		}
	}

	// Create decrypted message reader.
	if msgDetails, err = openpgp.ReadMessage(encReader, entityList, nil, nil); err != nil {
		return nil, err
	}

	return msgDetails.UnverifiedBody, nil
}
