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

	res := make(map[string]bool)
	for _, pkg := range args {
		// Look up the package ID
		key, err := repolib.GetPkgKey(clear_version, pkg)
		if err != nil {
			log.Fatal(err)
		}

		// Build a file list for the specific package
		files, err := repolib.GetFiles(clear_version, key)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			res[f] = true
		}
	}
	for f := range res {
		fmt.Println(f)
	}
}
