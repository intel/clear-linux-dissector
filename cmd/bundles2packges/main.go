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
	"os"
	"regexp"
	"strings"
	"net/http"
)

func name_from_header(header string, version int) string {
	re := regexp.MustCompile(`clr-bundles-[1-9].*/bundles/(.*)`)
	match := re.FindStringSubmatch(header)
	if len(match) == 0 {
		return ""
	}
	return match[len(match) - 1]
}

func dump_package_deps(g *gographviz.Graph, p string, visited map[string]bool) []string {
	var ret []string;
	visited[p] = true
	for edge_name, edge := range g.Edges.SrcToDsts {
		if edge_name == p {
			for destination, _ := range edge {
				for relation_name, relation := range g.Relations.ParentToChildren {
					if destination == relation_name {
						for dname, _ := range relation {
							if !visited[dname] {
								for _, pname := range dump_package_deps(g, dname, visited) {
									ret = append(ret, pname)
								}
							}
							ret = append(ret, dname)
						}
					}
				}
			}
		}
	}
	return ret
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
	if info.Mode() & os.ModeNamedPipe != 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			new_args := strings.Split(scanner.Text(), " ")
			args = append(args, new_args...)
		}
	}

	if (graph_filename == "") {
		fmt.Println("No dependency graph file provided")
		flag.Usage()
		os.Exit(-1)
	}

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
	for pkg, _ := range pkgs_before_deps {
		pkg = fmt.Sprintf("\"%s\"", pkg)
		if !visited[pkg] {
			dump_package_deps(graph, pkg, visited)
		}
	}
	for k, _ := range visited {
		fmt.Println(strings.Replace(k, "\"", "", 2))
	}
}
