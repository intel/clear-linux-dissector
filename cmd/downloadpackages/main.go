package main

import (
	"bufio"
	"flag"
	"fmt"
	"clr-dissector/internal/downloader"
	"log"
	"os"
	"strings"
)

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
		err := downloader.DownloadFile(target, url)
		if err != nil {
			log.Fatal(err)
		}
	}
}
