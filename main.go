package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

type httpResponse struct {
	status          string
	host            string
	responseHeaders map[string]string
	time            int64
}

var (
	// Command line flags.
	example         bool
	delay           int
	responseHeaders string
	useHttp         bool
	version         = "devel" // for -v flag, updated during the release process with -ldflags=-X=main.version=...
)

func init() {
	flag.BoolVar(&example, "example", false, "Print example usage")
	flag.BoolVar(&useHttp, "useHttp", false, "Default to HTTP instead of HTTPS")
	flag.IntVar(&delay, "delay", 0, "The time between in requests, in seconds")
	flag.StringVar(&responseHeaders, "responseHeaders", "", "Comma delimited list of response headers to return")
	flag.Usage = usage
}

func printExampleUsage() {
	fmt.Println("url-pinger https://www.google.com")
	fmt.Println("url-pinger -delay 2 https://www.google.com")
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] URL\n\n", os.Args[0])
	fmt.Println("***If you do not specify the protocol in the URL, we default to HTTPS")
	fmt.Fprintln(os.Stderr, "OPTIONS:")
	flag.PrintDefaults()
}

func main() {
	flag.Parse()

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	url := args[0]

	if example {
		printExampleUsage()
		os.Exit(0)
	}

	var count int
	fmt.Fprintln(writer, "Time\tCount\tUrl\tResult\tTime\tHeaders")
	fmt.Fprintln(writer, "-----\t-----\t---\t------\t----\t-------")
	for {
		url = parseURI(url)
		status, err := getResult(url)
		if err != nil {
			log.Fatal(err)
		}

		headerValues := handleMap(status.responseHeaders)
		fmt.Fprintf(writer, "[%v]\t[%d]\t[%s]\t[%s]\t[%dms]\t[%s]\n", time.Now().Format(time.RFC3339), count, url, status.status, status.time, headerValues)
		time.Sleep(time.Second * time.Duration(delay))
		count++
		writer.Flush()
	}
}

func parseURI(url string) string {
	if !strings.Contains(url, "://") {
		if useHttp {
			url = "http://" + url
		} else {
			url = "https://" + url
		}
	}
	return url
}

func getResult(url string) (httpResponse, error) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	start := time.Now()
	response, err := client.Get(url)
	end := time.Since(start).Milliseconds()
	if err != nil {
		return httpResponse{}, err
	}
	defer response.Body.Close()

	h := make(map[string]string)
	s := strings.Split(responseHeaders, ",")
	for _, value := range s {
		h[value] = response.Header.Get(value)
	}

	return httpResponse{
		status:          response.Status,
		host:            response.Header.Get("Host"),
		responseHeaders: h,
		time:            end,
	}, nil
}

func handleMap(m map[string]string) string {
	var result string

	for k, v := range m {

		if len(m) > 1 {
			result += fmt.Sprintf(" {%s:%s} ", k, v)
		} else {
			result += fmt.Sprintf(" %s:%s ", k, v)
		}
	}
	return result
}
