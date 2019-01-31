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

	var base_repo_url string
	flag.StringVar(&base_repo_url, "repo_url",
		"https://cdn.download.clearlinux.org",
		"Base URL downloading releases")

	var base_bundles_url string
	flag.StringVar(&base_bundles_url, "bundles_url",
		"https://github.com/clearlinux/clr-bundles",
		"Base URL for downloading release archives of clr-bundles")

	var download_all bool
	flag.BoolVar(&download_all, "all", false,
		"Download all sources for the release")

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

	err = repolib.DownloadRepo(clear_version, base_repo_url)
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
	if download_all {
		for _, srpm := range srpmMap {
			downloads[srpm] = fmt.Sprintf("%s/releases/%d/clear/source/SRPMS/%s",
				base_repo_url, clear_version, srpm)
		}
	} else {
		pkgs := make(map[string]bool)
		for _, target_bundle := range args {
			b, err := repolib.GetBundle(clear_version, target_bundle)
			if err != nil {
				log.Fatal(err)
			}

			for p := range b["AllPackages"].(map[string]interface{}) {
				pkgs[p] = true
			}
		}

		for p := range pkgs {
			if srpmMap[p] == "" {
				fmt.Printf("No mapping found for %s!\n", p)
				os.Exit(-1)
			}
			downloads[srpmMap[p]] = fmt.Sprintf("%s/releases/%d/clear/source/SRPMS/%s",
				base_repo_url, clear_version, srpmMap[p])
		}
	}

	// Downlaod the source rpms
	for fname, url := range downloads {
		target := fmt.Sprintf("%d/srpms/%s", clear_version, fname)
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

	// Unarchive the source rpms
	for fname := range downloads {
		archive := fmt.Sprintf("%d/srpms/%s", clear_version, fname)
		target := fmt.Sprintf("%d/source/%s", clear_version,
			strings.TrimSuffix(fname, ".src.rpm"))

		// Remove the version and release sections from the name
		l := strings.Split(target, "-")
		target = strings.Join(l[:len(l)-2], "-")

		if _, err := os.Stat(target); !os.IsNotExist(err) {
			continue
		}
		fmt.Printf("Extracting %s to %s...\n", archive, target)
		err = repolib.ExtractRpm(archive, target)
		if err != nil {
			log.Fatal(err)
		}
	}
}
