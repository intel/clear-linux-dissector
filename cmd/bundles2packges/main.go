package main

import (
	"archive/tar"
	"bufio"
	"clr-dissector/internal/common"
	"clr-dissector/internal/repolib"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
)

func name_from_header(header string, version int) string {
	re := regexp.MustCompile(`clr-bundles-[1-9].*/bundles/(.*)`)
	match := re.FindStringSubmatch(header)
	if len(match) == 0 {
		return ""
	}
	return match[len(match)-1]
}

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

	config_url := fmt.Sprintf("%s/archive/%d.tar.gz",
		base_url, clear_version)

	resp, err := http.Get(config_url)
	if err != nil {
		log.Fatal(err)

	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		fmt.Printf("clr-bundle release archive not found on server:\n%s\n",
			config_url)
		os.Exit(-1)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	bundles := make(map[string][]string)
	deps := make(map[string][]string)
	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatal(err)
		}

		if header == nil {
			continue
		}

		bundle_name := name_from_header(header.Name, clear_version)
		if bundle_name != "" {
			scanner := bufio.NewScanner(tr)
			for scanner.Scan() {
				l := scanner.Text()

				if len(l) == 0 || strings.HasPrefix(l, "#") {
					continue
				}

				re := regexp.MustCompile(`include\((.*)\)`)
				match := re.FindStringSubmatch(l)
				if len(match) == 2 {
					// depends on another bundle
					deps[bundle_name] = append(deps[bundle_name], match[1])
				} else {
					bundles[bundle_name] = append(bundles[bundle_name], l)
				}
			}
		}

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
