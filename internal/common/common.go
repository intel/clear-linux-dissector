package common

import (
	"bufio"
	"errors"
	"fmt"
	"os"
)

func GetInstalledVersion() (int, error) {
	var res int
	f, err := os.Open("/usr/lib/os-release")
	if err != nil {
		return res, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		_, err := fmt.Sscanf(line, "VERSION_ID=%d", &res)
		if err == nil {
			break
		}
	}

	if res == 0 {
		return res, errors.New("No installed version available!")
	}
	
	return res, nil
}
