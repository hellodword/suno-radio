package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hellodword/suno-radio/internal/common"
	"github.com/hellodword/suno-radio/internal/mp3toogg"
)

func main() {
	addr := flag.String("addr", ":3001", "")

	healthcheck := flag.String("healthcheck", "", "http://1.example.org,http://2.example.org")
	flag.Parse()
	if *healthcheck != "" {
		http.DefaultClient.Timeout = time.Second
		for _, healhealthcheckURL := range strings.Split(*healthcheck, ",") {
			if healhealthcheckURL == "" {
				continue
			}
			if _, err := http.Get(healhealthcheckURL); err != nil {
				os.Exit(1)
			}
		}
		os.Exit(0)
	}

	err := common.CheckFfmpeg()
	if err != nil {
		panic(err)
	}

	converter := &mp3toogg.MP3ToOgg{}
	err = rpc.Register(converter)
	if err != nil {
		panic(err)
	}

	rpc.HandleHTTP()
	log.Println("RPC listening on", *addr)
	l, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()

	var wg sync.WaitGroup
	var errC = make(chan error)

	server := &http.Server{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := server.Serve(l)
		if err != nil {
			if !errors.Is(err, http.ErrServerClosed) {
				errC <- err
			}
		}
	}()

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt, syscall.SIGTERM)
	select {
	case <-exit:
	case <-errC:
	}

	server.Close()
	wg.Wait()
}
