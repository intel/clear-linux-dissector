package common

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func GetInstalledVersion() (int, error) {
	var res int
	f, err := os.Open("/usr/lib/os-release")
	if err != nil {
		return res, err
	}
	defer f.Close()

	kmap := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		tokens := strings.Split(scanner.Text(), "=")
		if len(tokens) != 2 {
			continue
		}
		kmap[tokens[0]] = tokens[1]
	}

	// clear-linux-os
	if id, ok := kmap["ID"]; ok && id == "clear-linux-os" {
		if version, ok := kmap["VERSION_ID"]; ok {
			_, err := fmt.Sscanf(version, "%d", &res)
			if err == nil {
				return res, nil
			}
		}
	}

	return res, errors.New("No installed version available!")
}
