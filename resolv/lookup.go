package resolv

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/miekg/dns"
	goCache "github.com/pmylund/go-cache"
	"github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"
)

var (
	client     *dns.Client
	conn       *goCache.Cache
	saveSig    = make(chan os.Signal)
	file       = filepath.Join(path.Dir(os.Args[0]), "cache.dat")
	resolvConf = filepath.Join(path.Dir(os.Args[0]), "resolv.conf")
)

func init() {
	client = new(dns.Client)
	client.Net = "udp"
	conn = goCache.New(time.Second*time.Duration(60*60*12), time.Second*60)
	conn.LoadFile(file)
	intervalSaveCache()
	signal.Notify(saveSig, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGQUIT)
}

func intervalSaveCache() {
	save := func() {
		err := conn.SaveFile(file)
		if err == nil {
			logrus.Infof("cache saved: %s\n", file)
		} else {
			logrus.Errorf("cache save failed: %s, %s\n", file, err)
		}
	}
	go func() {
		for {
			select {
			case sig := <-saveSig:
				save()
				switch sig {
				case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
					os.Exit(0)
				}
			case <-time.After(time.Second * 180):
				save()
			}
		}
	}()
}

func toMd5(data string) string {
	m := md5.New()
	m.Write([]byte(data))
	return hex.EncodeToString(m.Sum(nil))
}

func DnsLookup(req *dns.Msg) (*dns.Msg, error) {
	var (
		key   string
		m     *dns.Msg
		err   error
		data  []byte
		id    uint16
		query []string
		// questions []dns.Question
		conf *dns.ClientConfig
	)
	conf, err = dns.ClientConfigFromFile(resolvConf)
	if err != nil {
		logrus.Error("read resolv file failure", err)
		panic(err)
	}

	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	query = make([]string, len(req.Question))
	for i, q := range req.Question {
		// if q.Qtype != dns.TypeAAAA || false {
		// 	questions = append(questions, q)
		// }
		req.SetQuestion(q.Name, dns.TypeA)
		query[i] = fmt.Sprintf("(%s %s %s)", q.Name, dns.ClassToString[q.Qclass], dns.TypeToString[q.Qtype])
	}
	// if len(questions) == 0 {
	// 	return nil, nil
	// }
	// req.Question = questions

	id = req.Id

	req.Id = 0
	key = toMd5(req.String())
	req.Id = id
	if reply, ok := conn.Get(key); ok {
		data, _ = reply.([]byte)
		if data != nil && len(data) > 0 {
			m = &dns.Msg{}
			_ = m.Unpack(data)
			m.Id = id
			return m, nil
		}
	}
	for _, server := range conf.Servers {
		server := fmt.Sprintf("%s:%s", server, conf.Port)
		m, err = lookup(req, client, server, false)
		if err == nil && len(m.Answer) > 0 {
			m.Id = 0
			data, _ = m.Pack()
			conn.Set(key, data, 0)
			m.Id = id
			return m, nil
		}
	}
	logrus.Error("dnslook end err", err)
	return nil, err
}
func lookup(msg *dns.Msg, client *dns.Client, server string, edns bool) (*dns.Msg, error) {
	if edns {
		opt := &dns.OPT{
			Hdr: dns.RR_Header{
				Name:   ".",
				Rrtype: dns.TypeOPT,
			},
		}
		opt.SetUDPSize(dns.DefaultMsgSize)
		msg.Extra = append(msg.Extra, opt)
	}
	response, _, err := client.Exchange(msg, server)
	if err != nil {
		logrus.Error("dns revels", err)
		return nil, err
	}

	if msg.Id != response.Id {
		logrus.Errorf("DNS ID mismatch, request: %d, response: %d", msg.Id, response.Id)
		return nil, fmt.Errorf("DNS ID mismatch, request: %d, response: %d", msg.Id, response.Id)
	}

	if response.MsgHdr.Truncated {
		if client.Net == "tcp" {
			logrus.Error("Got truncated message on tcp")
			return nil, fmt.Errorf("Got truncated message on tcp")
		}

		if edns { // Truncated even though EDNS is used
			client.Net = "tcp"
		}

		return lookup(msg, client, server, !edns)
	}

	return response, nil
}
