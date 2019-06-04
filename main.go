package main

import (
	"context"
	"fmt"
	"git.oriente.com/devops/dns-proxy/pb"
	"git.oriente.com/devops/dns-proxy/resolv"
	"github.com/miekg/dns"

	"google.golang.org/grpc"
	"log"
	"net"
)

type dnsServer struct{}

func (d *dnsServer) Query(ctx context.Context, in *pb.DnsPacket) (*pb.DnsPacket, error) {
	m := new(dns.Msg)
	if err := m.Unpack(in.Msg); err != nil {
		return nil, fmt.Errorf("failed to unpack msg: %v", err)
	}
	r := new(dns.Msg)
	r.SetReply(m)
	r.Authoritative = true
	for _, q := range r.Question {
		log.Println(q.Name)
		data, _ := resolv.Lookup(q.Qtype, q.Name)
		r.Answer = data.Answer
		log.Println(r.Answer)
	}

	if len(r.Answer) == 0 {
		r.Rcode = dns.RcodeNameError
	}

	out, err := r.Pack()
	if err != nil {
		return nil, fmt.Errorf("failed to pack msg: %v", err)
	}
	return &pb.DnsPacket{Msg: out}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":8053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("[info] --> listen: 8053 start")
	grpcServer := grpc.NewServer()
	pb.RegisterDnsServiceServer(grpcServer, &dnsServer{})
	panic(grpcServer.Serve(lis))
}
