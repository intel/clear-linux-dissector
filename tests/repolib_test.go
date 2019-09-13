package main

import (
	"fmt"
	"github.com/intel/clear-linux-dissector/internal/downloader"
	"github.com/intel/clear-linux-dissector/internal/repolib"
	"net/http"
	"testing"
)

func getCurrentVersion() (version int, err error) {
	resp, err := http.Get("https://cdn.download.clearlinux.org/current/latest")
	if err != nil {
		return 0, err
	}

	_, err = fmt.Fscanf(resp.Body, "%d", &version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

func TestRepoLib(t *testing.T) {
	version, err := getCurrentVersion()
	if err != nil {
		t.Fatal(err)
	}

	err = repolib.DownloadRepo(version,
		"https://cdn.download.clearlinux.org")
	if err != nil {
		t.Fatal(err)
	}

	_, err = repolib.GetBundle(version, "os-core")
	if err != nil {
		t.Fatal(err)
	}

	_, err = repolib.GetPkgMap(version)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repolib.GetSrpmHashMap(version)
	if err != nil {
		t.Fatal(err)
	}

	r := make(map[string]bool)
	r["libc6"] = true
	pkgs, err := repolib.QueryReqs(version, r, "rpm_sourcerpm")
	if err != nil {
		t.Fatal(err)
	}

	url := fmt.Sprintf("https://cdn.download.clearlinux.org/" +
		"releases/%d/clear/source/SRPMS/%s", version, pkgs[0])
	target := fmt.Sprintf("%d/srpms/%s", version, pkgs[0])
	err = downloader.DownloadFile(target, url, "", "")
	if err != nil {
		t.Fatal(err)
	}

	dst := fmt.Sprintf("%d/source/%s", version, pkgs[0])
	err = repolib.ExtractRpm(target, dst)
	if err != nil {
		t.Fatal(err)
	}
}
