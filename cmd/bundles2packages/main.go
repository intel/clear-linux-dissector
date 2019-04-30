package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/intel/clear-linux-dissector/internal/common"
	"github.com/intel/clear-linux-dissector/internal/repolib"
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

	pkgs, err := repolib.QueryReqs(clear_version, requirements, "name")
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range pkgs {
		fmt.Println(p)
	}
}
