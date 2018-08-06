package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"log"
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
	fmt.Printf("\rDownloading %s... %s complete", wc.Name, humanize.Bytes(wc.Total))
}

func DownloadFile(filepath string, url string) error {
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

	// download was successful so rename temporary file
	err = os.Rename(tmp, filepath)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var clear_version int
	flag.IntVar(&clear_version, "clear_version", -1, "Clear Linux version")

	var base_url string
	flag.StringVar(&base_url, "url",
		"https://cdn.download.clearlinux.org",
		"Base URL for downloading release source rpms")

	var mapping_file string
	flag.StringVar(&mapping_file, "mapping", "",
		"File containing mapping of binary to source rpm filenames")

	flag.Usage = func() {
		fmt.Printf("USAGE for %s\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()

	info, err := os.Stdin.Stat()
	if err != nil {
		log.Fatal()
	}
	if info.Mode()&os.ModeNamedPipe != 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			new_args := strings.Split(scanner.Text(), " ")
			args = append(args, new_args...)
		}
	}

	if clear_version == -1 {
		f, err := os.Open("/usr/lib/os-release")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := fmt.Sscanf(line, "VERSION_ID=%d", &clear_version)
			if err == nil {
				break
			}
		}
	}

	if mapping_file == "" {
		fmt.Println("No mapping file provided")
		flag.Usage()
		os.Exit(-1)
	}
	f, err := os.Open(mapping_file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	srpmMap := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		var bname string
		var sname string
		_, err := fmt.Sscanf(line, "%s %s", &bname, &sname)
		if err != nil {
			log.Fatal(err)
		}

		srpmMap[bname] = sname
	}

	downloads := make(map[string]string)
	for _, p := range args {
		if srpmMap[p] == "" {
			fmt.Printf("No mapping found for %s!\n", p)
			os.Exit(-1)
		}
		downloads[srpmMap[p]] = fmt.Sprintf("%s/releases/%d/clear/source/SRPMS/%s",
			base_url, clear_version, srpmMap[p])
	}

	// create cache directory if it doesn't already exist
	err = os.MkdirAll(".srpm_cache", 0700)
	if err != nil {
		log.Fatal(err)
	}
	for fname, url := range downloads {
		target := ".srpm_cache/" + fname
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			continue
		}
		err := DownloadFile(target, url)
		if err != nil {
			log.Fatal(err)
		}
	}
}
