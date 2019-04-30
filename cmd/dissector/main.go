package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/intel/clear-linux-dissector/internal/common"
	"github.com/intel/clear-linux-dissector/internal/downloader"
	"github.com/intel/clear-linux-dissector/internal/repolib"
	"log"
	"os"
	"strings"
	"io/ioutil"
	"strconv"
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

	if download_all {
		// Find most recent version subdir with downloaded SRPMs
		files, err := ioutil.ReadDir("./")
		if err != nil {
			log.Fatal(err)
		}
		maxoldver := 0
		for _, f := range files {
			if err == nil && f.IsDir() {
				if vernum, err := strconv.Atoi(f.Name()); err == nil && vernum < clear_version && vernum > maxoldver {
					spath := fmt.Sprintf("%s/srpms/.done", f.Name())
					_, err := os.Stat(spath)
					if err == nil {
						maxoldver = vernum
					}
				}
			}
		}
		if maxoldver > 0 {
			// Grab any previously downloaded SRPMs whose sha256sums match
			olddir := fmt.Sprintf("%d", maxoldver)
			for _, srpm := range srpmMap {
				oldpth := fmt.Sprintf("%s/srpms/%s", olddir, srpm)
				_, err := os.Stat(oldpth)
				if err == nil {
					newpth := fmt.Sprintf("%d/srpms/%s", clear_version, srpm)
					_, err := os.Stat(newpth)
					if os.IsNotExist(err) {
						if hashmap[srpm] == "" {
							fmt.Printf("No hash found for %s!\n", srpm)
							os.Exit(-1)
						}
						actual_checksum, err := downloader.ChecksumFile(oldpth)
						if err != nil {
							continue
						}
						if actual_checksum == hashmap[srpm] {
							fmt.Printf("Copying previously downloaded %s\n", srpm)
							os.Link(oldpth, newpth)
						}
					}
				}
			}
		}
	}

	downloads := make(map[string]string)
	if download_all {
		for _, srpm := range srpmMap {
			downloads[srpm] = fmt.Sprintf("%s/releases/%d/clear/source/SRPMS/%s",
				base_repo_url, clear_version, srpm)
		}
	} else {
		requirements := make(map[string]bool)
		for _, target_bundle := range args {
			b, err := repolib.GetBundle(clear_version, target_bundle)
			if err != nil {
				log.Fatal(err)
			}

			for p := range b["AllPackages"].(map[string]interface{}) {
				requirements[p] = true
			}
		}

		pkgs, err := repolib.QueryReqs(clear_version, requirements, "rpm_sourcerpm")
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range pkgs {
			downloads[p] = fmt.Sprintf("%s/releases/%d/clear/source/SRPMS/%s",
				base_repo_url, clear_version, p)
		}
	}

	// Download the source rpms
	i := 0
	dlcount := len(downloads)
	for fname, url := range downloads {
		i++
		target := fmt.Sprintf("%d/srpms/%s", clear_version, fname)
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			continue
		}
		if hashmap[fname] == "" {
			fmt.Printf("No hash found for %s!\n", fname)
			os.Exit(-1)
		}
		extra := fmt.Sprintf("(%d/%d) ", i, dlcount)
		err := downloader.DownloadFile(target, url, hashmap[fname], extra)
		if err != nil {
			log.Fatal(err)
		}
	}

	if download_all {
		// We're done downloading srpms, mark the directory as done
		dotfpath := fmt.Sprintf("%d/srpms/.done", clear_version)
		f, err := os.OpenFile(dotfpath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
	}

	// Unarchive the source rpms
	i = 0
	for fname := range downloads {
		i++
		archive := fmt.Sprintf("%d/srpms/%s", clear_version, fname)
		target := fmt.Sprintf("%d/source/%s", clear_version,
			strings.TrimSuffix(fname, ".src.rpm"))

		// Remove the version and release sections from the name
		l := strings.Split(target, "-")
		target = strings.Join(l[:len(l)-2], "-")

		if _, err := os.Stat(target); !os.IsNotExist(err) {
			continue
		}
		fmt.Printf("Extracting (%d/%d) %s to %s...\n", i, dlcount, archive, target)
		err = repolib.ExtractRpm(archive, target)
		if err != nil {
			log.Fatal(err)
		}
	}
}
