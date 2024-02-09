package listen

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/evocert/lnksnk/concurrent"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var listerens = concurrent.NewMap()

func ShutdownAll() {
	listerens.Range(func(key, value any) bool {
		if lstnr, _ := value.(*http.Server); lstnr != nil {
			lstnr.Shutdown(context.Background())
			fmt.Println("Shutdown - ", key)
		}
		return true
	})
}

func Shutdown(keys ...interface{}) {
	if len(keys) > 0 {
		keys = append(keys, func(delkeys []interface{}, delvalues []interface{}) {
			for kn, k := range delkeys {
				if lstnr, _ := delvalues[kn].(*http.Server); lstnr != nil {
					lstnr.Shutdown(context.Background())
					fmt.Println("Shutdown - ", k)
				}
			}
		})
		listerens.Del(keys...)
	}
}

type listen struct {
	handler   http.Handler
	TLSConfig *tls.Config
}

func (lsnt *listen) Serve(network string, addr string, tlsconf ...*tls.Config) {
	if lsnt != nil {
		Serve(network, addr, lsnt.handler, tlsconf...)
	}
}

func (lsnt *listen) ServeTLS(network string, addr string, orgname string, tlsconf ...*tls.Config) {
	if lsnt != nil {
		rsvldaddr, _ := net.ResolveTCPAddr(func() string {
			if network == "quic" {
				return "tcp"
			}
			return network
		}(), addr)

		host := rsvldaddr.IP.String()
		certhost := host
		if host == "" || host == "<nil>" {
			host = "localhost"
			certhost = host
		}

		if addrnames, _ := net.LookupHost(host); len(addrnames) > 0 {
			for _, adr := range addrnames {
				host = adr
			}
		}
		if len(tlsconf) == 0 {
			if publc, prv, err := GenerateTestCertificate(certhost, orgname); err == nil {
				if cert, err := tls.X509KeyPair(publc, prv); err == nil {
					tslcnf := &tls.Config{InsecureSkipVerify: true}
					tslcnf.Certificates = append(tslcnf.Certificates, cert)
					tlsconf = append(tlsconf, tslcnf)
				}
			}
		}
		Serve(network, addr, lsnt.handler, tlsconf...)
	}
}

func (lsnt *listen) Shutdown(keys ...interface{}) {
	if lsnt != nil {
		Shutdown(keys...)
	}
}

func NewListen(handerfunc http.HandlerFunc) *listen {
	if handerfunc == nil && DefaultHandler != nil {
		handerfunc = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			DefaultHandler.ServeHTTP(w, r)
		})
	}
	return &listen{handler: handerfunc, TLSConfig: &tls.Config{}}
}

func Serve(network string, addr string, handler http.Handler, tlsconf ...*tls.Config) {
	if handler == nil && DefaultHandler != nil {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			DefaultHandler.ServeHTTP(w, r)
		})
	}
	if strings.Contains(network, "tcp") {

		if handler != nil {
			if ln, err := Listen(network, addr); err == nil { //net.Listen(network, addr); err == nil {

				if tlsconfL := len(tlsconf); tlsconfL > 0 && tlsconf[0] != nil {
					ln = tls.NewListener(ln, tlsconf[0].Clone())
				}

				httpsrv := &http.Server{Handler: h2c.NewHandler(handler, &http2.Server{})}
				listerens.Set(ln.Addr().String(), httpsrv)
				go httpsrv.Serve(ln)
				return
			}
		}
	} else if strings.Contains(network, "quic") {
		if len(tlsconf) > 0 {
			server := http3.Server{
				Handler:    handler,
				Addr:       addr,
				TLSConfig:  http3.ConfigureTLSConfig(tlsconf[0].Clone()), // use your tls.Config here
				QuicConfig: &quic.Config{},
			}
			go server.ListenAndServe()
		}
		//http3.ListenAndServeQUIC(addr, "/path/to/cert", "/path/to/key", handler)
	}
}

var DefaultHandler http.Handler = nil

var httpsrv = &http.Server{Handler: h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	if DefaultHandler != nil {
		DefaultHandler.ServeHTTP(w, r)
	}
}), &http2.Server{})}

func init() {
	go func() {
		//httpsrv.Serve(lstnr)
	}()
}

// GenerateTestCertificate generates a test certificate and private key based on the given host.
func GenerateTestCertificate(host string, orgname string) ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, err
	}

	if orgname == "" {
		orgname = "LNKSNK"
	}
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{orgname},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		DNSNames:              []string{host},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certBytes, err := x509.CreateCertificate(
		rand.Reader, cert, cert, &priv.PublicKey, priv,
	)

	p := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		},
	)

	b := pem.EncodeToMemory(
		&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		},
	)

	return b, p, err
}
