package main

import (
	"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/dustin/go-humanize"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

type Pkgs struct {
	XMLName  xml.Name `xml:"repomd"`
	Data     []Data   `xml:"data"`
}

type Data struct {
	XMLName      xml.Name     `xml:"data"`
	Type         string       `xml:"type,attr"`
	Location     Location     `xml:"location"`
	Checksum     Checksum     `xml:"checksum"`
	OpenChecksum OpenChecksum `xml:"open-checksum"`
}

type Location struct {
	XMLName xml.Name `xml:"location"`
	Href    string   `xml:"href,attr"`
}

type Checksum struct {
	XMLName xml.Name `xml:"checksum"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:",chardata"`
}

type OpenChecksum struct {
	XMLName xml.Name `xml:"open-checksum"`
	Type    string   `xml:"type,attr"`
	Value   string   `xml:",chardata"`
}

type WriteCounter struct {
	Total uint64
	Name string
}
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}
func (wc WriteCounter) PrintProgress() {
	fmt.Printf("\r%s", strings.Repeat(" ", 80))
	fmt.Printf("\rDownloading %s... %s complete", wc.Name, humanize.Bytes(wc.Total))
}

func DownloadFile(filepath string, url string) error {
	if _, err := os.Stat(filepath); !os.IsNotExist(err) {
		return nil
	}

	tmp := filepath + ".tmp"

	// temporary file
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	counter := &WriteCounter{Name: filepath}
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return err
	}
	
	// Clear the progress output
	fmt.Print("\n")

	// download was successful so rename temporary file
	err = os.Rename(tmp, filepath)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	var clear_version int
	flag.IntVar(&clear_version, "clear_version", -1, "Clear Linux version")

	var base_url string
	flag.StringVar(&base_url, "url",
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
	if info.Mode() & os.ModeNamedPipe != 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			new_args := strings.Split(scanner.Text(), " ")
			args = append(args, new_args...)
		}
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

	config_url := fmt.Sprintf(
		"%s/releases/%d/clear/x86_64/os/repodata/repomd.xml",
		base_url, clear_version)

	resp, err := http.Get(config_url)
	if err != nil {
		log.Fatal(err)
		
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	if (resp.Status != "200 OK") {
		fmt.Printf("Unable to find release %d on %s",
			clear_version, base_url)
		os.Exit(-1)
	}

	path := fmt.Sprintf("%d/repodata", clear_version)
	err = os.MkdirAll(path, 0700)
	if err != nil {
		log.Fatal(err)
	}
	
	
	var pkgs Pkgs
	xml.Unmarshal(body, &pkgs)
	for i := 0; i < len(pkgs.Data); i++ {
		href := pkgs.Data[i].Location.Href
		url := fmt.Sprintf(
			"%s/releases/%d/clear/x86_64/os/%s",
			base_url, clear_version, href)

		if strings.HasSuffix(href, "other.xml.gz") {
			t := fmt.Sprintf("%d/repodata/other.xml.gz", clear_version)
			err := DownloadFile(t, url)
			if err != nil {
				log.Fatal(err)
			}
		} else if strings.HasSuffix(href, "primary.xml.gz") {
			t := fmt.Sprintf("%d/repodata/primary.xml.gz", clear_version)
			err := DownloadFile(t, url)
			if err != nil {
				log.Fatal(err)
			}
		} else if strings.HasSuffix(href, "comps.xml.xz") {
			t := fmt.Sprintf("%d/repodata/comps.xml.xz", clear_version)
			err := DownloadFile(t, url)
			if err != nil {
				log.Fatal(err)
			}
		} else if strings.HasSuffix(href, "filelists.xml.gz") {
			t := fmt.Sprintf("%d/repodata/filelist.xml.gz", clear_version)
			err := DownloadFile(t, url)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
