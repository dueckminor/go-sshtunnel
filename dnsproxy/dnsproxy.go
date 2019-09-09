package dnsproxy

import (
	"fmt"

	"github.com/miekg/dns"
)

func ForwardDNS(listenAddr, targetAddr string) {
	fmt.Printf("Forward DNS requests to: %s\n", targetAddr)

	dns.HandleFunc(".", func(w dns.ResponseWriter, r *dns.Msg) {
		switch r.Opcode {
		case dns.OpcodeQuery:
			dnsClient := new(dns.Client)
			dnsClient.Net = "tcp"
			// fmt.Println("----- REQUEST -----")
			// fmt.Println(r)
			response, _, err := dnsClient.Exchange(r, targetAddr)
			if err != nil {
				fmt.Println(err)
			}
			// fmt.Println("----- RESPONSE -----")
			// fmt.Println(response)
			w.WriteMsg(response)
		}
	})
	server := &dns.Server{Addr: listenAddr, Net: "udp"}
	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
