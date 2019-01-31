package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.intel.com/crlynch/clr-dissector/internal/common"
	"github.intel.com/crlynch/clr-dissector/internal/downloader"
	"github.intel.com/crlynch/clr-dissector/internal/repolib"
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

	var skip_download bool
	flag.BoolVar(&skip_download, "skip", false,
		"Skip downloading any source rpm files")

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
		clear_version, err = common.GetInstalledVersion()
		if err != nil {
			fmt.Println("A version must be specified when not " +
				"running on a Clear Linux instance!")
			os.Exit(-1)
		}
	}

	// Download repo data if needed and initialize directory structure
	err = repolib.DownloadRepo(clear_version, base_url)
	if err != nil {
		log.Fatal(err)
	}

	// Query db for map of binary to source packages
	srpmMap, err := repolib.GetPkgMap(clear_version)
	if err != nil {
		log.Fatal(err)
	}

	// Query source package db for map of srpm to sha256
	hashmap, err := repolib.GetSrpmHashMap(clear_version)
	if err != nil {
		log.Fatal(err)
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

	for fname, url := range downloads {
		target := fmt.Sprintf("%d/source/%s", clear_version, fname)
		if skip_download == true {
			fmt.Printf("Skipping %s\n", url)
			continue
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			continue
		}
		if hashmap[fname] == "" {
			fmt.Printf("No hash found for %s!\n", fname)
			os.Exit(-1)
		}
		err := downloader.DownloadFile(target, url, hashmap[fname])
		if err != nil {
			log.Fatal(err)
		}
	}
}
