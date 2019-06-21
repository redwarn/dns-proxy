package main

import (
	"context"
	"fmt"
	"git.oriente.com/devops/dns-proxy/pb"
	"git.oriente.com/devops/dns-proxy/resolv"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
	"runtime"

	"google.golang.org/grpc"
	"net"
)

type dnsServer struct{}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU()*2 - 1)
}
func (d *dnsServer) Query(ctx context.Context, in *pb.DnsPacket) (*pb.DnsPacket, error) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Error("Query this panic ", err)
		}
	}()

	m := new(dns.Msg)
	if err := m.Unpack(in.Msg); err != nil {
		return nil, fmt.Errorf("failed to unpack msg: %v", err)
	}
	r := new(dns.Msg)
	r.SetReply(m)
	r.Authoritative = true
	data, err := resolv.DnsLookup(m)
	if err != nil {
		logrus.Errorf("failed to query: %v", err)
		return nil, fmt.Errorf("failed to query: %v", err)
	}
	r.Answer = data.Answer
	logrus.Info(r.Answer[len(r.Answer)-1])
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
		logrus.Error("failed to listen: %v", err)
	}
	logrus.Info("listen: 8053 start")
	grpcServer := grpc.NewServer()
	pb.RegisterDnsServiceServer(grpcServer, &dnsServer{})
	panic(grpcServer.Serve(lis))
}
