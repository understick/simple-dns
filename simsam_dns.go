package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.comcom/miekg/dns"
)

type Config struct {
	ZoneFile  string
	Port      int
	Forwarder string
}

type Server struct {
	config  *Config
	records map[string][]dns.RR
}

func NewServer(serverConfig *Config) (*Server, error) {
	dnsServer := &Server{
		config:  serverConfig,
		records: make(map[string][]dns.RR),
	}

	err := dnsServer.loadZoneFile()
	if err != nil {
		return nil, fmt.Errorf("could not load zone file: %v", err)
	}

	return dnsServer, nil
}

// loadZoneFile opens and reads the zone file
func (server *Server) loadZoneFile() error {
	file, err := os.Open(server.config.ZoneFile)
	if err != nil {
		return err
	}
	defer file.Close()

	parser := dns.NewZoneParser(file, "", server.config.ZoneFile)

	for rr, ok := parser.Next(); ok; rr, ok = parser.Next() {
		key := fmt.Sprintf("%s:%s", strings.ToLower(rr.Header().Name), dns.TypeToString[rr.Header().Rrtype])

		server.records[key] = append(server.records[key], rr)
		log.Printf("Loaded record: %s", rr.String())
	}

	if err := parser.Err(); err != nil {
		return err
	}

	log.Printf("Loaded %d record keys from zone file", len(server.records))
	return nil
}

func (server *Server) handleRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		log.Printf("Received query for: %s type: %s", q.Name, dns.TypeToString[q.Qtype])

		key := fmt.Sprintf("%s:%s", strings.ToLower(q.Name), dns.TypeToString[q.Qtype])

		records, found := server.records[key]

		if found {
			log.Printf("Found local record for %s", key)
			m.Answer = append(m.Answer, records...)
		} else if server.config.Forwarder != "" {
			log.Printf("No local record. Forwarding query to %s", server.config.Forwarder)

			answers, err := server.forwardRequest(q)
			if err != nil {
				log.Printf("Error forwarding request: %v", err)
				m.SetRcode(r, dns.RcodeServerFailure)
			} else {
				m.Authoritative = false
				m.Answer = append(m.Answer, answers...)
			}
		}
	}

	err := w.WriteMsg(m)
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// sends query to upstream DNS server.
func (server *Server) forwardRequest(q dns.Question) ([]dns.RR, error) {
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion(q.Name, q.Qtype)
	m.RecursionDesired = true

	response, _, err := c.Exchange(m, server.config.Forwarder)
	if err != nil {
		return nil, err
	}

	return response.Answer, nil
}

// starts DNS server and waits for a signal to shut down.
func (server *Server) Run() {

	dns.HandleFunc(".", server.handleRequest)
	go func() {
		srv := &dns.Server{Addr: ":" + strconv.Itoa(server.config.Port), Net: "udp"}
		log.Printf("DNS server %s starting on port %d", version, server.config.Port)
		if err := srv.ListenAndServe(); err != nil {
			log.Fatalf("Failed to start UDP server: %s", err.Error())
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	log.Fatalf("Signal (%v) received, stopping server.", s)
}

func main() {
	serverConfig := &Config{}

	flag.StringVar(&serverConfig.ZoneFile, "zone_file", "./zone.txt", "Path to the zone file.")
	flag.IntVar(&serverConfig.Port, "port", 53, "Port to listen on.")
	flag.StringVar(&serverConfig.Forwarder, "server", "8.8.8.8:53", "Upstream DNS server. Leave empty to disable forwarding.")
	flag.Parse()

	log.Printf("Starting server with config: %+v", *serverConfig)

	dnsServer, err := NewServer(serverConfig)
	if err != nil {
		log.Fatalf("Error creating server: %v", err)
	}

	dnsServer.Run()
}
