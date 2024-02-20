package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		panic(err)
	}

	const N = 100
	var serverData [N]int
	var clientData [N]int

	s := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bts, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			n, err := strconv.Atoi(string(bts))
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			serverData[n]++
			w.Write([]byte("hello world"))
		}),
	}

	go func() {
		fmt.Println("http.Server.Serve", s.Serve(l))
	}()

	timeout := time.Second * 2
	c := http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2:   false,
			MaxIdleConnsPerHost: 1000,
		},
		Timeout: timeout,
	}
	addr := l.Addr().String()
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for {
				rsp, err := c.Post(
					"http://"+addr+"/",
					"Application-json",
					bytes.NewReader([]byte(strconv.Itoa(i))))
				if err != nil {
					if strings.Contains(err.Error(), "connection refused") {
						return
					}
					fmt.Println("Post", err)
					return
				} else if rsp.StatusCode != http.StatusOK {
					fmt.Println("Post", rsp.StatusCode)
					return
				}

				clientData[i]++
				if _, err = io.ReadAll(rsp.Body); err != nil {
					fmt.Println("read body error", err)
					return
				}
				rsp.Body.Close()
			}
		}(i)
	}

	time.Sleep(timeout * 3)
	fmt.Println("http.Server.Shutdown", s.Shutdown(context.Background()))
	wg.Wait()

	for i := 0; i < N; i++ {
		if clientData[i] != serverData[i] {
			fmt.Printf("data at index %d mismatch, client %d, server %d\n", i, clientData[i], serverData[i])
		}
	}
}
