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

	// Compile a complete list of packages from both the
	// requested bundles and the bundle included from
	// those bundles.
	//
	// This does not include additional packages that each
	// of these packages depend on
	pkgs := make(map[string]bool)
	for _, target_bundle := range args {
		if _, ok := bundles[target_bundle]; !ok {
			if _, ok := deps[target_bundle]; !ok {
				fmt.Printf("Bundle %s does not exist!\n",
					target_bundle)
				os.Exit(-1)
			}
		}
		for _, p := range bundles[target_bundle] {
			pkgs[p] = true
		}
		for _, b := range deps[target_bundle] {
			for _, p := range bundles[b] {
				pkgs[p] = true
			}
		}
	}
	for p := range pkgs {
		fmt.Println(p)
	}
}
