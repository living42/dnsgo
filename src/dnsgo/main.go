package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/miekg/dns"
)

func main() {
	conf, err := GetConfig()
	if err != nil {
		fmt.Println(err)
		return
	}

	if conf.cpuprofile != "" {
		f, err := os.Create(conf.cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("starting cpuprofile")
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	if conf.memprofile != "" {
		log.Println("starting memprofile")
		defer func() {
			f, err := os.Create(conf.memprofile)
			if err != nil {
				log.Fatal(err)
			}
			pprof.WriteHeapProfile(f)
		}()
	}

	dns.HandleFunc(".", handleQuestion)
	if conf.ListenTCP {
		go serve("tcp", conf.Listen)
	}
	if conf.ListenUDP {
		go serve("udp", conf.Listen)
	}
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

forever:
	for {
		select {
		case s := <-sig:
			fmt.Printf("Signal (%v) received, stopping\n", s)
			break forever
		}
	}
}

func handleQuestion(w dns.ResponseWriter, m *dns.Msg) {
	conf, _ := GetConfig()

	log.Printf("question %s %s from %s\n", m.Question[0].Name, dns.TypeToString[m.Question[0].Qtype], w.RemoteAddr())

	resultch := make(chan *dns.Msg)

	for _, resolver := range conf.Resolvers {
		go resolver.Resolve(m, resultch)
	}

	r := <-resultch
	r.SetRcode(m, r.Rcode)
	w.WriteMsg(r)
}

func serve(net string, listen string) {
	server := &dns.Server{Addr: listen, Net: net, TsigSecret: nil}
	log.Printf("%s server listen on %s\n", net, listen)
	err := server.ListenAndServe()
	if err != nil {
		log.Printf("Failed to setup the "+net+" server: %s\n", err.Error())
	}
}
