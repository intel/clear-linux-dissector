package repolib

import (
	"clr-dissector/internal/downloader"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"io/ioutil"
	"os"
	"strings"
)

type Pkgs struct {
	XMLName xml.Name `xml:"repomd"`
	Data    []Data   `xml:"data"`
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

func DownloadRepo(version int, url string) error {
	config_url := fmt.Sprintf(
		"%s/releases/%d/clear/x86_64/os/repodata/repomd.xml",
		url, version)

	resp, err := http.Get(config_url)
	if err != nil {
		return err

	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.Status != "200 OK" {
		return errors.New(fmt.Sprintf("Unable to find release %d on %s",
			version, url))
	}

	path := fmt.Sprintf("%d/repodata", version)
	err = os.MkdirAll(path, 0700)
	if err != nil {
		return err
	}

	var pkgs Pkgs
	xml.Unmarshal(body, &pkgs)
	for i := 0; i < len(pkgs.Data); i++ {
		href := pkgs.Data[i].Location.Href
		url := fmt.Sprintf(
			"%s/releases/%d/clear/x86_64/os/%s",
			url, version, href)

		if strings.HasSuffix(href, "other.xml.gz") {
			t := fmt.Sprintf("%d/repodata/other.xml.gz", version)
			err := downloader.DownloadFile(t, url)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(href, "primary.xml.gz") {
			t := fmt.Sprintf("%d/repodata/primary.xml.gz", version)
			err := downloader.DownloadFile(t, url)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(href, "comps.xml.xz") {
			t := fmt.Sprintf("%d/repodata/comps.xml.xz", version)
			err := downloader.DownloadFile(t, url)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(href, "filelists.xml.gz") {
			t := fmt.Sprintf("%d/repodata/filelist.xml.gz", version)
			err := downloader.DownloadFile(t, url)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
