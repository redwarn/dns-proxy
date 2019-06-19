package main

import (
	"git.oriente.com/devops/dns-proxy/resolv"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

func main() {
	dns.HandleFunc(".", proxyServe)

	failure := make(chan error, 1)

	go func(failure chan error) {
		failure <- dns.ListenAndServe(":53", "tcp", nil)
	}(failure)

	go func(failure chan error) {
		failure <- dns.ListenAndServe(":53", "udp", nil)
	}(failure)

	logrus.Info("ready for accept connection on tcp/udp :53 ...\n",)

	logrus.Error(<-failure)
}

func proxyServe(w dns.ResponseWriter, req *dns.Msg) {
	data,err := resolv.DnsLookup(req)
	if err != nil {
		logrus.Error(err)
		err = w.WriteMsg(nil)
		if err != nil {
			logrus.Error(err)
		}
	}
	err = w.WriteMsg(data)
	if err != nil {
		logrus.Error(err)
	}
}
