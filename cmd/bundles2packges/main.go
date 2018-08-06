package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"flag"
	"fmt"
	"github.com/awalterschulze/gographviz"
	"io"
	"io/ioutil"
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

func dump_package_deps(g *gographviz.Graph, p string, visited map[string]bool) []string {
	var res []string
	visited[p] = true

	// Walk dependency graph from source to destinations,
	// recursing on each destination package the first time
	// it is seen
	for emap_key, emap := range g.Edges.SrcToDsts {
		if emap_key == p {
			for edge_key := range emap {
				for rmap_key, rmap := range g.Relations.ParentToChildren {
					if rmap_key == edge_key {
						for dname := range rmap {
							if !visited[dname] {
								// recurse over newly uncovered packages to resolve additional deps
								for _, pname := range dump_package_deps(g, dname, visited) {
									res = append(res, pname)
								}
							}
							res = append(res, dname)
						}
					}
				}
			}
		}
	}
	return res
}

func main() {
	var clear_version int
	flag.IntVar(&clear_version, "clear_version", -1, "Clear Linux version")

	var base_url string
	flag.StringVar(&base_url, "url",
		"https://github.com/clearlinux/clr-bundles",
		"Base URL for downloading release archives of clr-bundles")

	var graph_filename string
	flag.StringVar(&graph_filename, "dependency_graph", "",
		"Input dependency graph file")

	var dump_all bool
	flag.BoolVar(&dump_all, "dump_all", false, "Dump all bundles")

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

	if graph_filename == "" {
		fmt.Println("No dependency graph file provided")
		flag.Usage()
		os.Exit(-1)
	}

	if clear_version == -1 {
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

	f, err := os.Open(graph_filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	content, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}

	graph, err := gographviz.Read(content)
	if err != nil {
		log.Fatal(err)
	}

	visited := make(map[string]bool)
	if dump_all {
		for _, v := range bundles {
			for _, vv := range v {
				visited[vv] = true
			}
		}
	} else {
		for pkg := range pkgs_before_deps {
			pkg = fmt.Sprintf("\"%s\"", pkg)
			if !visited[pkg] {
				dump_package_deps(graph, pkg, visited)
			}
		}
	}
	for k := range visited {
		fmt.Println(strings.Replace(k, "\"", "", 2))
	}
}
