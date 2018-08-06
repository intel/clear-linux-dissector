package repolib

import (
	"encoding/xml"
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
