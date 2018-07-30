package main

import (
	"archive/tar"
	"compress/gzip"
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"net/http"
	"regexp"
)

func name_from_header(header string, version int) string {
	re := regexp.MustCompile(`clr-bundles-[1-9].*/bundles/(.*)`)
	match := re.FindStringSubmatch(header)
	if len(match) == 0 {
		return ""
	}
	return match[len(match) - 1]
}

func main() {
	var clear_version int
	flag.IntVar(&clear_version, "v", -1, "Clear Linux version")

	var base_url string
	flag.StringVar(&base_url, "u",
		"https://github.com/clearlinux/clr-bundles",
		"Base URL for downloading release archives of clr-bundles")
	
	flag.Usage = func() {
		fmt.Printf("USAGE for %s\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if (clear_version == -1) {
		f, err := os.Open("/usr/lib/os-release")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := fmt.Sscanf(line, "VERSION_ID=%d", &clear_version)
			if err == nil {
				break
			}
		}
	}
	
	config_url := fmt.Sprintf("%s/archive/%d.tar.gz",
		base_url, clear_version)

	resp, err := http.Get(config_url)
	if err != nil {
		log.Fatal(err)
		
	}
	defer resp.Body.Close()

	if (resp.Status != "200 OK") {
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
			break;
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

	res := make(map[string]bool)
	for _, target_bundle := range flag.Args() {
		for _, p := range bundles[target_bundle] {
			res[p] = true
		}
		for _, b := range deps[target_bundle] {
			for _, p := range bundles[b] {
				res[p] = true
			}			
		}
	}
	for p, _ := range res {
	 	fmt.Println(p)
	}
}
