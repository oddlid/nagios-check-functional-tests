package main

import (
	"encoding/xml"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"time"
)

const (
	VERSION   string = "2017-02-06"
	UA        string = "VGT MnM ApiCheck/1.0"
	DEF_TMOUT int    = 10
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

	if timeout == nil {
		timeout = time.Second * time.Duration(DEF_TMOUT)
	}

	client := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	return client.Do(req)
}

func parse() {
}

func entryPoint(ctx *cli.Context) error {
	file, err := os.Open("sample.xml")
	if err != nil {
		log.Error(err)
		return err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Error(err)
		return err
	}

	cr := CheckResponse{}
	err = xml.Unmarshal(data, &cr)
	if err != nil {
		log.Errorf("Unable to parse XML: %q", err)
		return err
	}

	log.Debugf("%#v", cr)

	//out, err := xml.MarshalIndent(cr, "", "\t")
	//if err != nil {
	//	log.Error(err)
	//}
	//os.Stdout.Write(out)

	fmt.Printf("Pretty format:\n%s\n", cr.String())
	fmt.Printf("All OK: %t\n", cr.Ok())

	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "check_vgtapi"
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
