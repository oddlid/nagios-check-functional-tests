package main

import (
	"crypto/tls"
	"encoding/xml"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const (
	VERSION    string  = "2017-02-07"
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

// getUrl() fetches a URL and returns the HTTP response
func getUrl(url string, verifySSL bool, timeout time.Duration, ua string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	if ua == "" {
		req.Header.Set("User-Agent", UA)
	} else {
		req.Header.Set("User-Agent", ua)
	}

	tr := &http.Transport{
		DisableKeepAlives: true, // we're (probably) not reusing the connection, so don't let it hang open
	}

	if !verifySSL {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	return client.Do(req)
}

func parse(url string, timeout time.Duration, ch chan CheckResponse) {
	cr := CheckResponse{URL: url}
	t_start := time.Now()
	resp, err := getUrl(url, false, timeout, "")
	cr.ResponseTime = time.Now().Sub(t_start)
	if err != nil {
		log.Error(err)
		cr.Err = err
		ch <- cr
		return
	}
	defer resp.Body.Close()
	cr.HTTPCode = resp.StatusCode

	if cr.HTTPCode != http.StatusOK {
		ch <- cr
		return
	}

	cr.Body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		cr.Err = err
		ch <- cr
		return
	}

	err = xml.Unmarshal(cr.Body, &cr)
	if err != nil {
		log.Error(err)
		cr.Err = err
	}

	ch <- cr
}

func entryPoint(ctx *cli.Context) error {
	url := ctx.String("url")
	verbose := ctx.Bool("verbose")
	warn := ctx.Float64("warning")
	crit := ctx.Float64("critical")
	to := ctx.Float64("timeout")
	tmout := time.Second * time.Duration(to)

	log.WithFields(log.Fields{
		"url":      url,
		"verbose":  verbose,
		"warning":  warn,
		"critical": crit,
		"timeout":  to,
	}).Debug("Entrypoint params")

	chres := make(chan CheckResponse)
	defer close(chres)

	go parse(url, tmout, chres)

	// helper func for printing results and exiting
	_e := func(ecode int, desc string, cr *CheckResponse) {
		perfstr := fmt.Sprintf("|time=%fs;%f;%f\n", cr.ResponseTime.Seconds(), warn, crit)
		var status string
		switch ecode {
		case E_OK:
			status = S_OK
		case E_WARNING:
			status = S_WARNING
		case E_CRITICAL:
			status = S_CRITICAL
		default:
			status = S_UNKNOWN
		}
		msg := fmt.Sprintf("%s: %s; Response time: %f; URL: %q%s", status, desc, cr.ResponseTime.Seconds(), url, perfstr)
		if verbose {
			msg += cr.String()
		}
		fmt.Println(msg)
		os.Exit(ecode)
	}

	select {
	case res := <-chres:
		if res.Err != nil {
			_e(E_CRITICAL, fmt.Sprintf("%q", res.Err.Error()), &res)
		}
		if res.HTTPCode != http.StatusOK {
			_e(E_UNKNOWN, fmt.Sprintf("HTTP problem, code: %d", res.HTTPCode), &res)
		}
		if !res.Ok() {
			_e(E_CRITICAL, "Response tagged as failed, see long output", &res)
		}
		if res.ResponseTime.Seconds() >= crit {
			_e(E_CRITICAL, "Response time at or above critical limit", &res)
		}
		if res.ResponseTime.Seconds() >= warn {
			_e(E_WARNING, "Response time at or above warning limit", &res)
		}
		//fmt.Printf("%s\n", res.String())
		_e(E_OK, "All good", &res)
	case <-time.After(tmout):
		fmt.Printf("%s: Timed out after %.2f seconds getting %q\n", S_UNKNOWN, to, url)
		os.Exit(E_UNKNOWN)
	}


	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "check_functional_tests"
	app.Version = VERSION
	//app.Compiled, _ = time.Parse(time.RFC3339, BUILD_DATE)
	app.Usage = "Nagios Functional Check for VGT API"
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "Odd E. Ebbesen",
			Email: "odd.ebbesen@wirelesscar.com",
		},
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "url, U",
			Usage: "URL to check",
		},
		cli.Float64Flag{
			Name:  "timeout, t",
			Usage: "Timeout in seconds, fractions allowed",
			Value: DEF_TMOUT,
		},
		cli.Float64Flag{
			Name:  "warning, w",
			Usage: "Warning responsetime in seconds, fractions allowed",
			Value: DEF_WARN,
		},
		cli.Float64Flag{
			Name:  "critical, c",
			Usage: "Critical responsetime in seconds, fractions allowed",
			Value: DEF_CRIT,
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "Print long output",
		},
		cli.StringFlag{
			Name:  "log-level, l",
			Value: "error",
			Usage: "Log level (options: debug, info, warn, error, fatal, panic).",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "Run in debug mode.",
		},
	}

	app.Before = func(ctx *cli.Context) error {
		level, err := log.ParseLevel(ctx.String("log-level"))
		if err != nil {
			log.Fatal(err.Error())
		}
		log.SetLevel(level)
		if !ctx.IsSet("log-level") && !ctx.IsSet("l") && ctx.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}
		log.SetFormatter(&log.TextFormatter{
			DisableTimestamp: false,
			FullTimestamp:    true,
		})

		return nil
	}

	app.Action = entryPoint
	app.Run(os.Args)
}
