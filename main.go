package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

type httpResponse struct {
	status          string
	host            string
	responseHeaders map[string]string
	latency         int64
}

var (
	// Command line flags.
	example         bool
	delay           int
	responseHeaders string
	useHTTP         bool
	version         = "devel" // for -v flag, updated during the release process with -ldflags=-X=main.version=...
)

func init() {
	flag.BoolVar(&example, "example", false, "Print example usage")
	flag.BoolVar(&useHTTP, "usehttp", false, "Default to HTTP instead of HTTPS")
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

	if example {
		printExampleUsage()
		os.Exit(0)
	}

	url := args[0]
	if !strings.Contains(url, "://") {
		url = parseURI(url)
	}

	var count int = 0
	var responses []httpResponse
	var httpErr error

	fmt.Fprintln(writer, "Time\tCount\tUrl\tResult\tTime\tHeaders")
	fmt.Fprintln(writer, "-----\t-----\t---\t------\t----\t-------")

	// capture a ctrl+c event, to print out statistics at the end
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Printf("Total Requests: %d\n", count-1)
		os.Exit(0)
	}()

	for {
		responses, httpErr = getResult(url)
		if httpErr != nil {
			log.Fatal(httpErr)
		}

		for _, status := range responses {
			headerValues := handleMap(status.responseHeaders)
			fmt.Fprintf(writer, "[%v]\t[%d]\t[%s]\t[%s]\t[%dms]\t[%s]\n", time.Now().Format(time.RFC3339), count, url, status.status, status.latency, headerValues)
			count++
		}

		time.Sleep(time.Second * time.Duration(delay))
		writer.Flush()
	}
}

func parseURI(url string) string {
	if !useHTTP {
		return "https://" + url
	}
	return "http://" + url
}

func getResult(url string) ([]httpResponse, error) {

	var responses []httpResponse

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	start := time.Now()
	response, err := client.Get(url)
	end := time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	h := make(map[string]string)
	s := strings.Split(responseHeaders, ",")
	for _, value := range s {
		h[value] = response.Header.Get(value)
	}

	r := httpResponse{
		status:          response.Status,
		host:            response.Header.Get("Host"),
		responseHeaders: h,
		latency:         end,
	}
	responses = append(responses, r)
	return responses, nil
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
