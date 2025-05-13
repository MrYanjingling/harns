package main

import (
	"fmt"
	"io"
	"lightiot/pkg/client"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func main() {
	u, _ := url.Parse("http://192.168.1.119")
	tp := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	c := client.NewClient(u, 60*time.Second, tp)
	location, _ := time.LoadLocation("Asia/Shanghai")
	thingId := "5faddba380aa460497061385c85c11e0"
	start := time.Date(2021, time.May, 18, 0, 0, 0, 0, location)
	end := time.Date(2021, time.June, 18, 0, 0, 0, 0, location)

	begin := time.Now()
	fmt.Printf("begin: %s\n", begin)
	var wg sync.WaitGroup
	for i := 1; i <= 1500; i++ {
		go getTs(&wg, c, thingId, fmt.Sprintf("tala%d", i), start, end)
	}

	wg.Wait()
	stop := time.Now()
	fmt.Printf("stop: %s, duration: %s\n", stop, stop.Sub(start))
}

func getTs(wg *sync.WaitGroup, c client.Client, thingId, ps string, start, end time.Time) {
	wg.Add(1)
	defer wg.Done()
	for start.Before(end) {
		go func(wg *sync.WaitGroup, start time.Time) {
			wg.Add(1)
			defer wg.Done()
			q := &url.Values{
				"start": []string{start.Format(time.RFC3339Nano)},
				"end":   []string{start.Add(24 * time.Hour).Format(time.RFC3339Nano)},
			}
			resp, err := c.Get(fmt.Sprintf("/api/data/v1/timeseries/%s/%s", thingId, ps), nil, q)
			if err != nil {
				fmt.Println(err)
			}
			io.Copy(io.Discard, resp.Body)
			defer resp.Body.Close()
		}(wg, start)

		start = start.Add(24 * time.Hour)
	}
}
