package main

import (
	"bufio"
	"flag"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"net/http"
)

func main() {
	var clear_version int
	flag.IntVar(&clear_version, "v", -1, "Clear Linux version")

	var image_name string
	flag.StringVar(&image_name, "n", "", "Name of Clear Linux image")

	var base_url string
	flag.StringVar(&base_url, "u", "https://cdn.download.clearlinux.org/releases",
		"Base URL for Clear repository")
	
	flag.Usage = func() {
		fmt.Printf("USAGE for %s\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()


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
	
	config_url := fmt.Sprintf("%s/%d/clear/config/image/%s-config.json",
		base_url, clear_version, image_name)

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
		fmt.Printf("Image \"%s\" for version %d was not found on the server\n",
			image_name, clear_version)
		os.Exit(-1)
	}
	
	var config map[string]interface{}
	json.Unmarshal(body, &config)
	bundles := config["Bundles"].([]interface{})

	s := make([]string, 0)
	for _, value := range bundles {
		s = append(s, value.(string))
	}
	fmt.Println(s)
}
