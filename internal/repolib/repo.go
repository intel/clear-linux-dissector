package repolib

import (
	"clr-dissector/internal/downloader"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"io"
	"io/ioutil"
	"os"
	"strings"

	_ "github.com/mutecomm/go-sqlcipher"
	"github.com/ulikunitz/xz"
)

type Repomd struct {
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

	err = os.MkdirAll(fmt.Sprintf("%d/repodata", version), 0700)
	if err != nil {
		return err
	}

	err = os.MkdirAll(fmt.Sprintf("%d/source", version), 0700)
	if err != nil {
		return err
	}

	var repomd Repomd
	xml.Unmarshal(body, &repomd)
	for i := 0; i < len(repomd.Data); i++ {
		href := repomd.Data[i].Location.Href
		url := fmt.Sprintf(
			"%s/releases/%d/clear/x86_64/os/%s",
			url, version, href)

		if strings.HasSuffix(href, "other.sqlite.xz") {
			t := fmt.Sprintf("%d/repodata/other.sqlite.xz", version)
			err := downloader.DownloadFile(t, url)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(href, "primary.sqlite.xz") {
			t := fmt.Sprintf("%d/repodata/primary.sqlite", version)
			err := downloader.DownloadFile(t+".xz", url)
			if err != nil {
				return err
			}
			err = UnXz(t+".xz", t)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(href, "comps.xml.xz") {
			t := fmt.Sprintf("%d/repodata/comps.xml.xz", version)
			err := downloader.DownloadFile(t, url)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(href, "filelists.sqlite.xz") {
			t := fmt.Sprintf("%d/repodata/filelist.sqlite.xz", version)
			err := downloader.DownloadFile(t, url)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func GetPkgMap(version int) (map[string]string, error) {
	pmap := make(map[string]string)
	db, err := sql.Open("sqlite3", fmt.Sprintf("%d/repodata/primary.sqlite",
		version))
	if err != nil {
		return pmap, err
	}
	defer db.Close()

	rows, err := db.Query("select name, rpm_sourcerpm from packages;")
	if err != nil {
		return pmap, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, srpm string
		err := rows.Scan(&name, &srpm)
		if err != nil {
			return pmap, nil
		}
		pmap[name] = srpm
	}
	

	return pmap, nil
}

func UnXz(gazin, gazout string) error {
	f, err := os.Open(gazin)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := xz.NewReader(f)
	if err != nil {
		return err
	}

	w, err := os.Create(gazout)
	if err != nil {
		return err
	}
	defer w.Close()
	
	if _, err = io.Copy(w, r); err != nil {
		return err
	}

	return nil
}

func getdeps(db *sql.DB, name string, visited map[string]bool) error {
	// Query list of requirements for the given package
	q := fmt.Sprintf("select requires.name from packages inner join requires " +
		"where packages.pkgKey=requires.pkgKey and packages.name='%s';", name)
	rows, err := db.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()

	rmap := make(map[string]bool)
	for rows.Next() {
		var rname string
		err := rows.Scan(&rname)
		if err != nil {
			return err
		}
		rmap[rname] = true
	}

	// Query list of packages that meet the found requirements
	pmap := make(map[string]bool)
	for p := range rmap {
		q := fmt.Sprintf("select packages.name from packages " +
			"inner join provides where packages.pkgKey=provides.pkgKey " +
			"and provides.name='%s';", p)
		rows, err := db.Query(q)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var pname string
			err := rows.Scan(&pname)
			if err != nil {
				return err
			}
			pmap[pname] = true
		}
	}
	
	for p := range pmap {
		if !visited[p] {
			visited[p] = true
			err := getdeps(db, p, visited)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func GetDirectDeps(name string, version int) ([]string, error) {
	db, err := sql.Open("sqlite3", fmt.Sprintf("%d/repodata/primary.sqlite",
		version))
	if err != nil {
		return nil, err
	}
	defer db.Close()

	visited := make(map[string]bool)
	getdeps(db, name, visited)

	var res []string
	for p, _ := range visited {
		res = append(res, p)
	}
	return res, nil
}

