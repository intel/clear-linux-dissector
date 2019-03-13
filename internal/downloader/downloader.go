package downloader

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"net/http"
	"os"
	"strings"
)

type WriteCounter struct {
	Total uint64
	Name  string
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}
func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 80))
	fmt.Printf("\rDownloading %s... %s complete", wc.Name,
		humanize.Bytes(wc.Total))
}

func ChecksumFile(filepath string) (string, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func DownloadFile(filepath, url, checksum string) error {
	if _, err := os.Stat(filepath); !os.IsNotExist(err) {
		return nil
	}

	tmp := filepath + ".tmp"

	// temporary file
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	counter := &WriteCounter{Name: filepath}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}

	// Clear the progress output
	fmt.Print("\n")

	if checksum != "" {
		actual_checksum, err := ChecksumFile(tmp)
		if err != nil {
			return err
		}
		if actual_checksum != checksum {
			os.Remove(tmp)
			return errors.New("Failed download checksum!")
		}
	}

	// download was successful so rename temporary file
	err = os.Rename(tmp, filepath)
	if err != nil {
		return err
	}

	return nil
}
