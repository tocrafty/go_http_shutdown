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

	wait := make(chan struct{})
	go func() {
		fmt.Println("ListenAndServe:8002",
			http.ListenAndServe(":8002", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				//fmt.Println("http.Server.Shutdown", s.Close())
				fmt.Println("http.Server.Shutdown", s.Shutdown(context.Background()))
				close(wait)
				w.WriteHeader(http.StatusOK)
			})))
	}()

	go func() {
		fmt.Println("http.Server.Serve", s.Serve(l))
	}()

	timeout := time.Second * 2
	addr := l.Addr().String()
	c := http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2:   false,
			MaxIdleConnsPerHost: 1000,
		},
		Timeout: timeout,
	}
	for i := 0; i < N; i++ {
		go func(i int) {
			for {
				rsp, err := c.Post(
					"http://"+addr+"/",
					"Application-json",
					bytes.NewReader([]byte(strconv.Itoa(i))))
				if err != nil {
					if strings.Contains(err.Error(), "connection refused") {
						return
					}
					fmt.Println("Get", err)
					return
				} else if rsp.StatusCode != http.StatusOK {
					fmt.Println("Get", rsp.StatusCode)
					return
				}
				if _, err = io.ReadAll(rsp.Body); err != nil {
					fmt.Println("read body error", err)
					return
				}
				clientData[i]++
				rsp.Body.Close()
			}
		}(i)
	}
	<-wait
	time.Sleep(timeout * 3)
	for i := 0; i < N; i++ {
		if clientData[i] != serverData[i] {
			fmt.Printf("data at index %d mismatch, client %d, server %d\n", i, clientData[i], serverData[i])
		}
	}
}
