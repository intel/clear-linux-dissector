package main

import (
	"bufio"
	"clr-dissector/internal/common"
	"clr-dissector/internal/repolib"
	"flag"
	"fmt"
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

	// use a map to remove duplicate entries
	results := make(map[string]bool)

	for _, pkg := range args {
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
