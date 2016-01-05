package main

import (
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/chop-dbhi/data-models-packer"
)

// createOutputWriter creates a writer based on a file if out is not empty or
// STDOUT if it is and returns a closer that must be closed.
func createOutputWriter(writer *io.Writer, out string) (closer io.Closer, err error) {

	if out != "" {

		outFile, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		*writer = outFile
		closer = outFile

	} else {

		*writer = os.Stdout
		closer = os.Stdout

	}

	return closer, err
}

// addEncryption creates an encrypting writer based on the passed writer using
// the file at key path if its passed and returns a closer that must be closed.
func addEncryption(writer *io.Writer, keyPath string) (closer io.Closer, err error) {

	// Default to encryption with PEDSNet DCC Public Key.
	keyReader := io.Reader(strings.NewReader(pedsnetDCCPublicKey))

	if keyPath != "" {

		keyReaderFile, err := os.Open(keyPath)
		if err != nil {
			return nil, err
		}

		defer keyReaderFile.Close()
		keyReader = io.Reader(keyReaderFile)
	}

	encWriter, err := packer.Encrypt(*writer, keyReader)
	if err != nil {
		return nil, err
	}

	*writer = encWriter
	return encWriter, nil
}

// addCompression creates a compressing writer based on the passed writer using
// the method specified in compr and returns a closer that must be closed.
func addCompression(writer *io.Writer, compr string) (closer io.Closer, err error) {

	switch compr {
	case "":
		compr = ".tar.gz"
	case ".tar.bzip2", ".tar.bz2":
		// bzip2 package does not implement a writer.
		return nil, errors.New("bzip2 compression not supported")
	case ".tar.gzip":
		compr = ".tar.gz"
	}

	compWriter, err := packer.NewCompressingWriter(*writer, compr)
	if err != nil {
		return nil, err
	}

	*writer = compWriter
	return compWriter, nil
}

// makeFilePacker returns a filepath.WalkFunc that packs files in the basePath
// directory using the passed writer.
func makeFilePacker(basePath string, w *packer.CompressingWriter) filepath.WalkFunc {

	return func(path string, fi os.FileInfo, inErr error) (err error) {

		var (
			relPath string
			r       *os.File
		)

		if err = inErr; err != nil {
			return err
		}

		if relPath, err = filepath.Rel(basePath, path); err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		if err = w.WriteHeader(fi, relPath); err != nil {
			return err
		}

		if r, err = os.Open(path); err != nil {
			return err
		}

		defer r.Close()

		log.Printf("packer: writing '%s' to package", fi.Name())

		if _, err = io.Copy(w, r); err != nil {
			return err
		}

		return nil
	}

}

const pedsnetDCCPublicKey = `
-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: SKS 1.1.5
Comment: Hostname: pgp.mit.edu

mQENBFTPy6gBCADO+bU5KritXPn5mh8PiUf2zDoNOVoxnoVsO1q/vLWmY+Dmk0Tf9CB15K7a
YgLRp++lL7fWqwGhCYitd3fi3lbhpDWJlVGoc3+pbYcivLwVHohJ/coP0sRsk8QlRgQZ9l6k
OxIUl1vRdnip3VVo3U+nuBmShcYDp4QY7s5/VMCrqHE6ho6KtunNUebclsUgGMEhoeWypk7Z
wZHPFIYmdK4E/3Ng6zDmIf1sFXeofi//MtNn8+cZLjaHQ0LFyNIgWdU2lOkNO9N5T3TDvhE5
ZXFsjqawrxVzTQZWZjk6FHrbQgavcIPXEr8JHsOoXE14BU3cE1TbOb8GpUVV/tDOUn/bABEB
AAG0UlBFRFNuZXQgRGF0YSBDb29yZGluYXRpbmcgQ2VudGVyIChEYXRhIFZhbGlkYXRpb24g
S2V5KSA8cGVkc25ldGRjY0BlbWFpbC5jaG9wLmVkdT6JATgEEwECACIFAlTPy6gCGwMGCwkI
BwMCBhUIAgkKCwQWAgMBAh4BAheAAAoJEBuu5RTrlyH35M8H/3PK6fC4h5gCXmAlSgqg19/i
4KBo9g4mQJKk4gUGZj0mEcPuq06/ycWTikYn77Hl5uaEXHGZRixPZ5jIv/Tf71DcorYFsStH
LEXPfIVjCbqWz74jKhf9K9/q7iZMNPx9JeVVDmBYZqwxZ4xHLEzlNXOcNywgXrhkc+Q9GnaN
yYOrwEXcIVpe6OYo9//UpKiglRySrxQqlRmO79BLr7sZydPAPfLlJXWOwTsbVR8jk6IM3Ad6
Cquv8KYHy6uCUryLG/rYai/0jtFb8+V93sJ8tJOf8XjoSY8NO7KWalkc4K2Iyc8g2IQ/bMIC
NmxuZz4GIgFRAumIWfz97CRxfiJgE7i5AQ0EVM/LqAEIAOTL0jAh56TyiSuC94qhV5ogwDrl
IuPs+Ck814pQBaV3n5Mo+CgPGRglPS+cc2M1XVJ5VA3NnSblmHmAV/Zw5/mUw/HIarEIs4P2
GwozhIOTRZ4fR1ZfaMhwWJY6hE4qxKqr4W7YsCABC/S0XsxVQRSfgqYzfYo9IwlHkLSRMBdd
3Z/cUWuMhGDe4Tm0T0phH3YF63sJJa9EAI8jcd4a5g1YSIquoACo1OSZM0RfZ5ZJtoYsFvTx
4uVoBqWhxL/w/mpkDHQzQFe2OIQ0vb7djTs3yzovuYwYvlUJFwP5Pv9kEYDA9bhz/KH4mOKl
UUfMVYd51yDlbixWFn39Xt68XYMAEQEAAYkBHwQYAQIACQUCVM/LqAIbDAAKCRAbruUU65ch
91LCCAC+3J7wm8gDkvlFNfDuOrlF6i/dy+x6tybZ6Ty1WJHx7ux6HJCv+fBORYWMQH0JhyXj
4hxSO7TjN416bz8OADTTaCDlh6GQCMpFrsBNakGbP7KMwQkFWhRW+hJvvUCrR7xlwYBYd7xK
2BWsVG8KdEp42NyZG/wKWsr2zaZ2xcY+SWYUxlq0fo/re2aNRMq9hnPDzaIb2TKA1AOfSDqA
9hpgrb5p6s5oJK5Mkrw3LGWSj0Ae0ovby4iU5cFShuo2JWaybw7J87YkhOQCkrTOoVQvOy/Q
iHCYOTfbjF9Gw4ZCa3Y9i+WgyKbGwqYCyLkT9Qf5VTx9qUp9/KbGRM0NGfW5
=LAW9
-----END PGP PUBLIC KEY BLOCK-----
`
