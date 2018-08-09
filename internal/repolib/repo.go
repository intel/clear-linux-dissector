package repolib

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"github.intel.com/crlynch/clr-dissector/internal/downloader"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
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

	err = os.MkdirAll(fmt.Sprintf("%d/srpms", version), 0700)
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
	q := fmt.Sprintf("select requires.name from packages inner join requires "+
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
		q := fmt.Sprintf("select packages.name from packages "+
			"inner join provides where packages.pkgKey=provides.pkgKey "+
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
	dbpath := fmt.Sprintf("%d/repodata/primary.sqlite", version)
	if _, err := os.Stat(dbpath); os.IsNotExist(err) {
		return nil, errors.New("Missing DB: " + dbpath)
	}

	db, err := sql.Open("sqlite3", dbpath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	visited := make(map[string]bool)
	getdeps(db, name, visited)

	var res []string
	for p := range visited {
		res = append(res, p)
	}
	return res, nil
}

func name_from_header(header string, version int) string {
	re := regexp.MustCompile(`clr-bundles-[1-9].*/bundles/(.*)`)
	match := re.FindStringSubmatch(header)
	if len(match) == 0 {
		return ""
	}
	return match[len(match)-1]
}

func GetBundles(clear_version int, base_url string) (map[string][]string, map[string][]string, error) {
	bundles := make(map[string][]string)
	deps := make(map[string][]string)

	config_url := fmt.Sprintf("%s/archive/%d.tar.gz",
		base_url, clear_version)

	resp, err := http.Get(config_url)
	if err != nil {
		return deps, bundles, err

	}
	defer resp.Body.Close()

	if resp.Status != "200 OK" {
		err := errors.New("clr-bundle release archive not found on server: " +
			config_url)
		return deps, bundles, err
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return deps, bundles, err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

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
					deps[bundle_name] = append(deps[bundle_name],
						match[1])
				} else {
					bundles[bundle_name] = append(bundles[bundle_name], l)
				}
			}
		}

	}

	return deps, bundles, nil
}
