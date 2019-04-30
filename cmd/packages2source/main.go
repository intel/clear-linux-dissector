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

	var base_repo_url string
	flag.StringVar(&base_repo_url, "repo_url",
		"https://cdn.download.clearlinux.org",
		"Base URL downloading releases")

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

	// Query db for map of binary to source packages
	srpmMap, err := repolib.GetPkgMap(clear_version)
	if err != nil {
		log.Fatal(err)
	}

	for _, p := range args {
		if srpmMap[p] == "" {
			fmt.Printf("No mapping found for %s!\n", p)
			os.Exit(-1)
		}
		fmt.Println(fmt.Sprintf("%s/releases/%d/clear/source/SRPMS/%s",
			base_repo_url, clear_version, srpmMap[p]))
	}
}
