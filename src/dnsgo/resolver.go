package main

import (
    "github.com/miekg/dns"
    "log"
    "time"
)

func (r *Resolver) resolveOne(server string, m *dns.Msg, result chan *dns.Msg, errch chan string) {
    q := m.Copy()
    q.Id = dns.Id()

    q.Compress = r.Compress

    client := new(dns.Client)
    client.Net = r.QueryMethod

    client.DialTimeout = time.Duration(r.Timeout) * time.Second
    client.WriteTimeout = time.Duration(r.Timeout) * time.Second
    client.ReadTimeout = time.Duration(r.Timeout) * time.Second

    if q.Compress {
        log.Printf("send to %s use %s [compressed]\n", server, client.Net)
    } else {
        log.Printf("send to %s use %s\n", server, client.Net)
    }
    a, _, err := client.Exchange(q, server)
    if err != nil {
        log.Println("exchange err:", err)
        errch <- server
        return
    }
    log.Printf("received from %s\n", server)
    result <- a
}

type Resolver struct {
    Server          []string          `yaml:"server"`
    Compress        bool              `yaml:"compress"`
    QueryMethod     string            `yaml:"query_method"`
    Timeout         int               `yaml:"timeout"`
    DomainPolicy    string            `yaml:"domain_policy"`
    Domain          []string          `yaml:"domain"`
    CountryPolicy   string            `yaml:"country_policy"`
    Country         []string          `yaml:"country`
}

func (r *Resolver) Resolve(m *dns.Msg, ch chan *dns.Msg) {

    // TODO: filter question
    result_ch := make(chan *dns.Msg)
    errch := make(chan string)

    for _, server := range r.Server {
        go r.resolveOne(server, m, result_ch, errch)
    }

    var err_count int

    for {
        select {
        case err_server := <-errch:
            err_count += 1
            log.Printf("%s timeout\n", err_server)
            if err_count >= len(r.Server) {
                log.Println("all server timeout")
                r := new(dns.Msg)
                r.SetRcode(m, dns.RcodeServerFailure)
                ch <- r
                return
            }
        case result := <- result_ch:
            // TODO: log server ip
            // TODO: log show all of TypeA answar
            // TODO: log answer type
            // TODO filter TypeA answar!!
           
            if r.pass(result) {
                ch <- result
                return
            }
        }
    }
}

func (r *Resolver) pass(m *dns.Msg) bool {
    conf, _ := GetConfig()
    for _, a := range m.Answer {
        switch a.(type){
        case *dns.A:
            for _, ccode := range r.Country {
                record, err := conf.GeoipDB.Country(a.(*dns.A).A)
                if err != nil {
                    log.Printf("parse country error for %s: %s", a.(*dns.A).A, err)
                    return false
                }
                if record.Country.IsoCode == ccode {
                    if r.CountryPolicy == POLICY_INCLUDED { 
                        log.Println(r.Server, "domain", a.(*dns.A).A, record.Country.IsoCode, "bypass")
                        return false 
                    } else if r.CountryPolicy == POLICY_EXCLUDED {
                        log.Println(r.Server, "domain", a.(*dns.A).A, record.Country.IsoCode, "pass")
                        return true
                    }
                } else {
                    if r.CountryPolicy == POLICY_INCLUDED { 
                        log.Println(r.Server, "domain", a.(*dns.A).A, record.Country.IsoCode, "pass")
                        return true 
                    } else if r.CountryPolicy == POLICY_EXCLUDED {
                        log.Println(r.Server, "domain", a.(*dns.A).A, record.Country.IsoCode, "bypass")
                        return false
                    }
                }
                log.Fatal("unknown country policy")
            }
        }
    }
    return true
}