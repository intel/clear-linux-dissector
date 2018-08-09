package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.intel.com/crlynch/clr-dissector/internal/common"
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

	deps, bundles, err := repolib.GetBundles(clear_version, base_url)
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
	results := make(map[string]bool)

	for pkg := range pkgs_before_deps {
		results[pkg] = true
		pkgs, err := repolib.GetDirectDeps(pkg, clear_version)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range pkgs {
			results[p] = true
		}
	}

	for p := range results {
		fmt.Println(p)
	}
}
