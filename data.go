package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type CheckResponse struct {
	XMLName      xml.Name     `xml:"CheckResponse"`
	MinorVersion int          `xml:"minorVersion"`
	Application  Applications `xml:"application"`
}

type Application struct {
	XMLName          xml.Name `xml:"application"`
	LongName         string   `xml:"longName"`
	ShortName        string   `xml:"shortName"`
	ComponentVersion string   `xml:"componentVersion"`
	Success          bool     `xml:"success"`
	FailureReason    string   `xml:"failureReason,omitempty"`
	Check            Checks   `xml:"check"`
}

type Check struct {
	Name          string `xml:"name"`
	Success       bool   `xml:"success"`
	FailureReason string `xml:"failureReason,omitempty"`
}

type Checks []Check
type Applications []Application

// _pp left-pads a string with <prefix> repeated <level> times,
// then right-pads the word <key> up to <align> length, then prints " : <val>\n"
func _pp(w io.Writer, prefix, key, val string, align, level int) {
	fmt.Fprintf(w, fmt.Sprintf("%s%s%d%s", strings.Repeat(prefix, level), "%-", align, "s : %s\n"), key, val)
	//pp_count++
}

func (c Check) pp(w io.Writer, prefix string, level int) {
	p := func(k, v string) {
		_pp(w, prefix, k, v, 13, level)
	}
	p("Name", c.Name)
	p("Success", fmt.Sprintf("%t", c.Success))
	if c.FailureReason != "" {
		p("FailureReason", c.FailureReason)
	}
}

func (a Application) pp(w io.Writer, prefix string, level int) {
	p := func(k, v string) {
		_pp(w, prefix, k, v, 16, level)
	}
	p("LongName", a.LongName)
	p("ShortName", a.ShortName)
	p("ComponentVersion", a.ComponentVersion)
	p("Success", fmt.Sprintf("%t", a.Success))
	if a.FailureReason != "" {
		p("FailureReason", a.FailureReason)
	}
	if len(a.Check) > 0 {
		for i := range a.Check {
			p("Check", "")
			a.Check[i].pp(w, prefix, level+1)
		}
	}
}

func (cr CheckResponse) pp(w io.Writer, prefix string, level int) {
	p := func(k, v string) {
		_pp(w, prefix, k, v, 12, level)
	}
	p("MinorVersion", fmt.Sprintf("%d", cr.MinorVersion))
	if len(cr.Application) > 0 {
		for i := range cr.Application {
			p("Application", "")
			cr.Application[i].pp(w, prefix, level+1)
		}
	}
}

func (cr CheckResponse) PrettyPrint(w io.Writer) {
	cr.pp(w, "  ", 0)
}

func (cr CheckResponse) String() string {
	var buf bytes.Buffer
	cr.PrettyPrint(&buf)
	return buf.String()
}

func (c Check) Ok() bool {
	return c.Success
}

func (c Checks) Ok() bool {
	if len(c) == 0 {
		return false
	}
	for i := range c {
		if !c[i].Ok() {
			return false
		}
	}
	return true
}

func (a Application) Ok() bool {
	return a.Success && a.Check.Ok()
}

func (a Applications) Ok() bool {
	if len(a) == 0 {
		return false
	}
	for i := range a {
		if !a[i].Ok() {
			return false
		}
	}
	return true
}

func (cr CheckResponse) Ok() bool {
	return cr.Application.Ok()
}
