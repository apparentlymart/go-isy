package isy

import (
	"encoding/xml"
)

type Function struct {
}

type Action interface{}

type triggersRaw struct {
	D2Ds []d2dRaw `xml:"d2d"`
}

type d2dRaw struct {
	Trigger triggerRaw `xml:"trigger"`
}

type triggerRaw struct {
	ID       int        `xml:"id"`
	Name     string     `xml:"name"`
	ParentID int        `xml:"parent"`
	IsFolder setBool    `xml:"folder"`
	Comment  string     `xml:"comment"`
	If       conditions `xml:"if"`
	Then     actionSeq  `xml:"then"`
	Else     actionSeq  `xml:"else"`
}

type actionSeq struct {
	Actions []Action
}

type conditions interface{}

type setBool bool

func (b *setBool) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// If the element is present at all then we're true
	*b = true
	return d.Skip()
}
