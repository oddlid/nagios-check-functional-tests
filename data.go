package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	VERSION    string  = "2017-02-16"
	UA         string  = "VGT MnM ApiCheck/1.0"
	DEF_INDENT string  = "  "
	DEF_TMOUT  float64 = 30.0
	DEF_WARN   float64 = 10.0
	DEF_CRIT   float64 = 15.0
	DEF_PORT   int     = 80
	E_OK       int     = 0
	E_WARNING  int     = 1
	E_CRITICAL int     = 2
	E_UNKNOWN  int     = 3
	S_OK       string  = "OK"
	S_WARNING  string  = "WARNING"
	S_CRITICAL string  = "CRITICAL"
	S_UNKNOWN  string  = "UNKNOWN"
)

type Checks []Check
type Applications []Application
type Keys []string

type CheckResponse struct {
	XMLName      xml.Name      `xml:"CheckResponse"`
	MinorVersion int           `xml:"minorVersion"`
	Application  Applications  `xml:"application"`
	ResponseTime time.Duration `xml:"-"`
	Body         []byte        `xml:"-"`
	HTTPCode     int           `xml:"-"`
	URL          string        `xml:"-"`
	Err          error         `xml:"-"`
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

func (k Keys) MaxLen() int {
	max := 0
	for i := range k {
		klen := len(k[i])
		if klen > max {
			max = klen
		}
	}
	return max
}

// _pp left-pads a string with <prefix> repeated <level> times,
// then right-pads the word <key> up to <align> length, then prints " : <val>\n"
func _pp(w io.Writer, prefix, key, val string, align, level int) {
	fmt.Fprintf(w, fmt.Sprintf("%s%s%d%s", strings.Repeat(prefix, level), "%-", align, "s : %s\n"), key, val)
	//pp_count++
}

// _hdr is a simplified version of _pp that is used for printing headers
func _hdr(w io.Writer, prefix, key, sep string, level int) {
	fmt.Fprintf(w, fmt.Sprintf("%s%s %s\n", strings.Repeat(prefix, level), key, sep))
}

func (c Check) pp(w io.Writer, prefix string, level int) {
	k := Keys{
		"Name",
		"Success",
		"FailureReason",
	}
	max := k.MaxLen()
	p := func(k, v string) {
		_pp(w, prefix, k, v, max, level)
	}
	p(k[0], c.Name)
	p(k[1], fmt.Sprintf("%t", c.Success))
	p(k[2], c.FailureReason)
}

func (a Application) pp(w io.Writer, prefix string, level int) {
	k := Keys{
		"LongName",
		"ShortName",
		"ComponentVersion",
		"Success",
		"FailureReason",
		"Check",
	}
	max := k.MaxLen()
	p := func(k, v string) {
		if v != "" {
			_pp(w, prefix, k, v, max, level)
		}
	}

	p(k[0], a.LongName)
	p(k[1], a.ShortName)
	p(k[2], a.ComponentVersion)
	p(k[3], fmt.Sprintf("%t", a.Success))
	p(k[4], a.FailureReason)

	if a.Check != nil {
		chklen := len(a.Check)
		if chklen > 0 {
			for i := range a.Check {
				_hdr(w, prefix, fmt.Sprintf("%s (#%d/%d)", k[5], i+1, chklen), "=>", level)
				a.Check[i].pp(w, prefix, level+1)
			}
		}
	}
}

func (cr CheckResponse) pp(w io.Writer, prefix string, level int) {
	k := Keys{
		"CheckResponse",
		"URL",
		"HTTP code",
		"Response time",
		"Error",
		"MinorVersion",
		"Application",
	}
	max := k.MaxLen()
	p := func(k, v string) {
		_pp(w, prefix, k, v, max, level)
	}

	fmt.Fprintf(w, "===== BEGIN: %s =====\n", k[0])
	p(k[1], cr.URL)
	p(k[2], fmt.Sprintf("%d", cr.HTTPCode))
	p(k[3], fmt.Sprintf("%f", cr.ResponseTime.Seconds()))
	if cr.Err != nil {
		p(k[4], cr.Err.Error())
	}
	p(k[5], fmt.Sprintf("%d", cr.MinorVersion))

	if cr.Application != nil {
		applen := len(cr.Application)
		if applen > 0 {
			for i := range cr.Application {
				_hdr(w, prefix, fmt.Sprintf("%s (#%d/%d)", k[6], i+1, applen), "=>", level)
				cr.Application[i].pp(w, prefix, level+1)
			}
		}
	}
	fmt.Fprintf(w, "===== END: %s =======\n", k[0])
}

func (cr CheckResponse) PrettyPrint(w io.Writer) {
	cr.pp(w, DEF_INDENT, 0)
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
