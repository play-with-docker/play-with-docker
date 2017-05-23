package handlers

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/play-with-docker/play-with-docker/config"
)

func DnsRequest(w dns.ResponseWriter, r *dns.Msg) {
	if len(r.Question) > 0 && config.NameFilter.MatchString(r.Question[0].Name) {
		// this is something we know about and we should try to handle
		question := r.Question[0].Name

		match := config.NameFilter.FindStringSubmatch(question)

		ip := strings.Replace(match[1], "-", ".", -1)

		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.RecursionAvailable = true
		a, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", question, ip))
		if err != nil {
			log.Fatal(err)
		}
		m.Answer = append(m.Answer, a)
		w.WriteMsg(m)
		return
	} else if len(r.Question) > 0 && config.AliasFilter.MatchString(r.Question[0].Name) {
		// this is something we know about and we should try to handle
		question := r.Question[0].Name

		match := config.AliasFilter.FindStringSubmatch(question)

		i := core.InstanceFindByAlias(match[2], match[1])

		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.RecursionAvailable = true
		a, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", question, i.IP))
		if err != nil {
			log.Fatal(err)
		}
		m.Answer = append(m.Answer, a)
		w.WriteMsg(m)
		return
	} else {
		if len(r.Question) > 0 {
			question := r.Question[0].Name

			if question == "localhost." {
				log.Printf("Not a PWD host. Asked for [localhost.] returning automatically [127.0.0.1]\n")
				m := new(dns.Msg)
				m.SetReply(r)
				m.Authoritative = true
				m.RecursionAvailable = true
				a, err := dns.NewRR(fmt.Sprintf("%s 60 IN A 127.0.0.1", question))
				if err != nil {
					log.Fatal(err)
				}
				m.Answer = append(m.Answer, a)
				w.WriteMsg(m)
				return
			}

			log.Printf("Not a PWD host. Looking up [%s]\n", question)
			ips, err := net.LookupIP(question)
			if err != nil {
				// we have no information about this and we are not a recursive dns server, so we just fail so the client can fallback to the next dns server it has configured
				w.Close()
				// dns.HandleFailed(w, r)
				return
			}
			log.Printf("Not a PWD host. Looking up [%s] got [%s]\n", question, ips)
			m := new(dns.Msg)
			m.SetReply(r)
			m.Authoritative = true
			m.RecursionAvailable = true
			for _, ip := range ips {
				ipv4 := ip.To4()
				if ipv4 == nil {
					a, err := dns.NewRR(fmt.Sprintf("%s 60 IN AAAA %s", question, ip.String()))
					if err != nil {
						log.Fatal(err)
					}
					m.Answer = append(m.Answer, a)
				} else {
					a, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", question, ipv4.String()))
					if err != nil {
						log.Fatal(err)
					}
					m.Answer = append(m.Answer, a)
				}
			}
			w.WriteMsg(m)
			return

		} else {
			log.Printf("Not a PWD host. Got DNS without any question\n")
			// we have no information about this and we are not a recursive dns server, so we just fail so the client can fallback to the next dns server it has configured
			w.Close()
			// dns.HandleFailed(w, r)
			return
		}
	}
}
