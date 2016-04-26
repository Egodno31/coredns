package dnssec

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"os"
	"time"

	"github.com/miekg/coredns/middleware"

	"github.com/miekg/dns"
)

type DNSKEY struct {
	K      *dns.DNSKEY
	s      crypto.Signer
	keytag uint16
}

// ParseKeyFile read a DNSSEC keyfile as generated by dnssec-keygen or other
// utilities. It adds ".key" for the public key and ".private" for the private key.
func ParseKeyFile(pubFile, privFile string) (*DNSKEY, error) {
	f, e := os.Open(pubFile)
	if e != nil {
		return nil, e
	}
	k, e := dns.ReadRR(f, pubFile)
	if e != nil {
		return nil, e
	}

	f, e = os.Open(privFile)
	if e != nil {
		return nil, e
	}
	p, e := k.(*dns.DNSKEY).ReadPrivateKey(f, privFile)
	if e != nil {
		return nil, e
	}

	if v, ok := p.(*rsa.PrivateKey); ok {
		return &DNSKEY{k.(*dns.DNSKEY), v, k.(*dns.DNSKEY).KeyTag()}, nil
	}
	if v, ok := p.(*ecdsa.PrivateKey); ok {
		return &DNSKEY{k.(*dns.DNSKEY), v, k.(*dns.DNSKEY).KeyTag()}, nil
	}
	return &DNSKEY{k.(*dns.DNSKEY), nil, 0}, errors.New("no known? private key found")
}

// getDNSKEY returns the correct DNSKEY to the client. Signatures are added when do is true.
func (d Dnssec) getDNSKEY(state middleware.State, zone string, do bool) *dns.Msg {
	keys := make([]dns.RR, len(d.keys))
	for i, k := range d.keys {
		keys[i] = dns.Copy(k.K)
		keys[i].Header().Name = zone
	}
	m := new(dns.Msg)
	m.SetReply(state.Req)
	m.Answer = keys
	if !do {
		return m
	}

	incep, expir := incepExpir(time.Now().UTC())
	if sigs, err := d.sign(keys, zone, 3600, incep, expir); err == nil {
		m.Answer = append(m.Answer, sigs...)
	}
	return m
}
