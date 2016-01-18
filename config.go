package packer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the configuration values for metadata creation and/or validation and package compression and/or decompression. The fields correspond exactly with the command line flags.
type Config struct {
	Comp         string
	DataVersion  string
	DataDirPath  string
	Etl          string
	KeyPassPath  string
	KeyPath      string
	Model        string
	ModelVersion string
	PackagePath  string
	Service      string
	Site         string
	VerifyOnly   bool
	packing      string
}

// Verify verifies the validity of a configuration.
func (cfg *Config) Verify() error {

	if cfg.PackagePath != "" {
		return cfg.handleExtensions()
	}

	return nil
}

// handleExtensions resolves the package path with the Config object.
// It sets the Comp and Encrypted attributes of the Config object based on the
// file path extensions, or errors if they conflict with pre-existing values.
func (cfg *Config) handleExtensions() error {

	var (
		name           string
		exts           map[string]bool
		j              int
		i              int
		cfgConflictErr error
		noTarErr       error
		extConflictErr error
		noCompErr      error
		noKeyErr       error
		noGpgErr       error
	)

	name = filepath.Base(cfg.PackagePath)

	// Read file extensions and normalize them into a map.
	exts = map[string]bool{
		"gpg": false,
		"tar": false,
		"bz2": false,
		"gz":  false,
		"zip": false,
	}

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
				return fmt.Errorf("unexpected file extension: %s\n", name[i+1:j])
			}
		}
	}

	// Resolve config with file extensions, throwing errors where appropriate.
	extConflictErr = errors.New("compression file extensions conflict with each other")
	noTarErr = errors.New("cannot use 'bz2' or 'gz' compression without 'tar'")
	cfgConflictErr = errors.New("file extensions conflict with passed compression method")
	noCompErr = errors.New("no compression extensions found")
	noKeyErr = errors.New("no key given for package with 'gpg' extension")
	noGpgErr = errors.New("no 'gpg' extension on package but key given")

	switch {

	case exts["gz"] && exts["bz2"] || exts["gz"] && exts["zip"] || exts["bz2"] && exts["zip"]:
		return extConflictErr

	case exts["gz"] && !exts["tar"] || exts["bz2"] && !exts["tar"]:
		return noTarErr

	case exts["tar"] && exts["gz"]:
		if cfg.Comp != "" && cfg.Comp != ".tar.gz" && cfg.Comp != ".tar.gzip" {
			return cfgConflictErr
		}
		cfg.Comp = ".tar.gz"

	case exts["tar"] && exts["bz2"]:
		if cfg.Comp != "" && cfg.Comp != ".tar.bz2" && cfg.Comp != ".tar.bzip2" {
			return cfgConflictErr
		}
		cfg.Comp = ".tar.bz2"

	case exts["zip"]:
		if cfg.Comp != "" && cfg.Comp != ".zip" {
			return cfgConflictErr
		}
		cfg.Comp = ".zip"

	default:
		return noCompErr
	}

	switch {

	case cfg.KeyPath == "" && exts["gpg"]:
		return noKeyErr

	case cfg.KeyPath != "" && !exts["gpg"]:
		return noGpgErr

	}

	return nil
}

// IsDir returns whether or not the passed path is a directory.
func IsDir(path string) (bool, error) {

	var (
		fi  os.FileInfo
		err error
	)

	// Stat input path to get file info.
	if fi, err = os.Stat(path); err != nil {
		return false, err
	}

	return fi.IsDir(), nil
}
