package packer

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/openpgp"
)

// CompressingWriter wraps zip.Writer and tar.Writer in a consistent API.
type CompressingWriter struct {

	// .tar.gz writers
	tarWriter  *tar.Writer
	gzipWriter *gzip.Writer

	// .zip compression fields
	zip       bool
	zipWriter *zip.Writer
	// zipFile stores each successive file object returned when a new header is
	// written to the zip.Writer.
	zipFile io.Writer
}

// NewCompressingWriter takes a writer to compress data onto and a string
// describing the compression method (".tar.gz" or ".zip") and returns a writer
// that can be written to and closed.
func NewCompressingWriter(comprWriter io.Writer, comp string) (writer *CompressingWriter, err error) {

	writer = new(CompressingWriter)

	if comp == ".zip" {
		writer.zip = true
		writer.zipWriter = zip.NewWriter(comprWriter)
		return writer, nil
	}

	// BUG(aaron0browne): The .tar.gz format compressed packages output by data
	// models packer cannot be read by standard tar and gzip tools.
	// Unfortunately, this is the default compression format and the only one
	// compatible with encryption. These packages can, of course, be unpacked
	// by data models packer.
	if comp == ".tar.gz" {
		writer.gzipWriter = gzip.NewWriter(comprWriter)
		writer.tarWriter = tar.NewWriter(writer.gzipWriter)
		return writer, nil
	}

	return nil, fmt.Errorf("unsupported compression method: %s", comp)

}

// WriteHeader writes a new file header and prepares to accept the file's
// contents.
func (w *CompressingWriter) WriteHeader(fi os.FileInfo, path string) (err error) {

	if w.zip {

		var zipHeader *zip.FileHeader

		// Create zip.Header from file info, adding the path.
		if zipHeader, err = zip.FileInfoHeader(fi); err != nil {
			return err
		}

		zipHeader.Name = path

		// Call the underlying zip.Writer method, storing the resulting file on
		// CompressingWriter.
		if w.zipFile, err = w.zipWriter.CreateHeader(zipHeader); err != nil {
			return err
		}

		return nil
	}

	var tarHeader *tar.Header

	// Create tar.Header from file info, adding the path
	if tarHeader, err = tar.FileInfoHeader(fi, ""); err != nil {
		return err
	}

	tarHeader.Name = path

	// Call the underlying tar.Writer method, which continues to use the same
	// writer.
	if err = w.tarWriter.WriteHeader(tarHeader); err != nil {
		return err
	}

	return nil
}

// Write writes data to the current entry in the archive.
func (w *CompressingWriter) Write(b []byte) (n int, err error) {

	if w.zip {
		return w.zipFile.Write(b)
	}

	return w.tarWriter.Write(b)
}

// Close closes the archive, flushing any unwritten data.
func (w *CompressingWriter) Close() (err error) {

	if w.zip {
		return w.zipWriter.Close()
	}

	// Close both tar and gzip writers if using ".tar.gz" compression.
	if err = w.tarWriter.Close(); err != nil {
		return err
	}

	return w.gzipWriter.Close()
}

// Encrypt takes a writer to encrypt data onto and a reader containing the
// ASCII-armored public key to encrypt with and returns a writer that can be
// written to and closed. It assumes there is only one OpenPGP entity involved.
func Encrypt(plainWriter io.Writer, keyReader io.Reader) (encWriter io.WriteCloser, err error) {

	entityList, err := openpgp.ReadArmoredKeyRing(keyReader)

	if err != nil {
		return nil, err
	}

	return openpgp.Encrypt(plainWriter, entityList, nil, nil, nil)
}
