package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestPack(t *testing.T) {
	if err := os.Mkdir("test_tmp", 0755); err != nil {
		t.Fatalf("packer tests: error making test temp dir: %s", err)
	}
	cmd := exec.Command("data-models-packer", "-keyPath", "test_data/key.asc", "-model", "pedsnet", "-modelVersion", "2.1.0", "-site", "ORG", "-etl", "https://specificanddurable.com/etlv3", "-out", "test_tmp/test.tar.gz.gpg", "test_data/data")
	if err := cmd.Run(); err != nil {
		t.Fatalf("packer tests: error using binary to pack: %s", err)
	}
	if err := os.Remove("test_data/data/metadata.csv"); err != nil {
		t.Fatalf("packer tests: error removing generated metadata.csv: %s", err)
	}
	if err := os.RemoveAll("test_tmp"); err != nil {
		t.Fatalf("packer tests: error removing test temp dir: %s", err)
	}
}

func TestUnpack(t *testing.T) {
	if err := os.Mkdir("test_tmp", 0755); err != nil {
		t.Fatalf("packer tests: error making test temp dir: %s", err)
	}
	cmd := exec.Command("data-models-packer", "-keyPath", "test_data/key.asc", "-keyPassPath", "test_data/pass.txt", "-out", "test_tmp", "test_data/test.tar.gz.gpg")
	if err := cmd.Run(); err != nil {
		t.Fatalf("packer tests: error testing unpacking with binary: %s", err)
	}
	if err := os.RemoveAll("test_tmp"); err != nil {
		t.Fatalf("packer tests: error removing test temp dir: %s", err)
	}
}

func TestVerify(t *testing.T) {
	cmd := exec.Command("data-models-packer", "-verifyOnly", "test_data")
	if err := cmd.Run(); err != nil {
		t.Fatalf("packer tests: error testing metadata verification with binary: %s", err)
	}
}
