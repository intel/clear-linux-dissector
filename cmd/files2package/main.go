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

	fmap, err := repolib.GetFilesMap(clear_version)
	if err != nil {
		log.Fatal(err)
	}

	keys := make(map[int]bool)
	for _, needle := range args {
		for key, pathmap := range fmap {
			if pathmap[needle] == true {
				keys[key] = true
			}
		}
	}

	for key := range keys {
		name, err := repolib.GetPkgName(clear_version, key)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(name)
	}
}
