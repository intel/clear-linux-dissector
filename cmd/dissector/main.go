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

	"github.com/sassoftware/go-rpmutils"
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

	deps, bundles, err := repolib.GetBundles(clear_version, base_bundles_url)
	if err != nil {
		log.Fatal(err)
	}

	pkgs_before_deps := make(map[string]bool)
	for _, target_bundle := range args {
		for _, p := range bundles[target_bundle] {
			pkgs_before_deps[p] = true
		}
		for _, b := range deps[target_bundle] {
			for _, p := range bundles[b] {
				pkgs_before_deps[p] = true
			}
		}
	}

	// use a map to remove duplicate entries
	pkgs := make(map[string]bool)

	for pkg := range pkgs_before_deps {
		pkgs[pkg] = true
		deps_for_pkg, err := repolib.GetDirectDeps(pkg, clear_version)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range deps_for_pkg {
			pkgs[p] = true
		}
	}

	// Query db for map of binary to source packages
	srpmMap, err := repolib.GetPkgMap(clear_version)
	if err != nil {
		log.Fatal(err)
	}

	downloads := make(map[string]string)
	for p := range pkgs {
		if srpmMap[p] == "" {
			fmt.Printf("No mapping found for %s!\n", p)
			os.Exit(-1)
		}
		downloads[srpmMap[p]] = fmt.Sprintf("%s/releases/%d/clear/source/SRPMS/%s",
			base_repo_url, clear_version, srpmMap[p])
	}

	// Downlaod the source rpms
	for fname, url := range downloads {
		target := fmt.Sprintf("%d/srpms/%s", clear_version, fname)
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			continue
		}
		err := downloader.DownloadFile(target, url)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Unarchive the source rpms
	for fname := range downloads {
		f, err := os.Open(fmt.Sprintf("%d/srpms/%s", clear_version, fname))
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		rpm, err := rpmutils.ReadRpm(f)
		if err != nil {
			log.Fatal(err)
		}

		target := fmt.Sprintf("%d/source/%s", clear_version,
			strings.TrimSuffix(fname, ".src.rpm"))
		err = rpm.ExpandPayload(target)
		if err != nil {
			log.Fatal(err)
		}
	}
}
