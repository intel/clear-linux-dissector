package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"github.com/awalterschulze/gographviz"
)


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
	var filename string
	flag.StringVar(&filename, "f", "", "Input dependency graph file")
	
	var list_packages bool
	flag.BoolVar(&list_packages, "list", false, "List all packages")
	
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

	if (filename == "") {
		fmt.Println("No dependency graph file provided")
		flag.Usage()
		os.Exit(-1)
	}
	
	f, err := os.Open(filename)
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

	if (list_packages) {
		for pname, _ := range graph.Nodes.Lookup {
			fmt.Println(strings.Replace(pname, "\"", "", 2))
		}
	}

	visited := make(map[string]bool)
	for _, pkg := range args {
		pkg = fmt.Sprintf("\"%s\"", pkg)
		if !visited[pkg] {
			dump_package_deps(graph, pkg, visited)
			for k, _ := range visited {
				fmt.Println(strings.Replace(k, "\"", "", 2))
			}
		}
	}
}
