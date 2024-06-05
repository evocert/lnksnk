package listen

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"golang.org/x/crypto/ocsp"
)

type OcspHandler struct {
	crt                tls.Certificate
	isRevoked          bool
	ocspNextUpdate     time.Time
	cachedOcspResponse []byte
	leaf, issuer       *x509.Certificate
	client             *http.Client
}

func (h *OcspHandler) getResponse() ([]byte, error) {
	ocspReq, err := ocsp.CreateRequest(h.leaf, h.issuer, nil)
	if err != nil {
		return nil, err
	}
	ocspReqBase64 := base64.StdEncoding.EncodeToString(ocspReq)

	reqURL := h.leaf.OCSPServer[0] + "/" + ocspReqBase64
	httpReq, err := http.NewRequest("GET", reqURL, nil)
	httpReq.Header.Add("Content-Language", "application/ocsp-request")
	httpReq.Header.Add("Accept", "application/ocsp-response")

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	barOcspResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ocspRes, err := ocsp.ParseResponse(barOcspResp, h.issuer)
	if err != nil {
		return nil, err
	}

	h.isRevoked = ocspRes.Status == ocsp.Revoked
	if ocspRes.Status == ocsp.Good {
		h.ocspNextUpdate = ocspRes.NextUpdate
		h.cachedOcspResponse = barOcspResp
		return barOcspResp, nil
	} else {
		h.ocspNextUpdate = time.Time{}
		h.cachedOcspResponse = nil
	}

	return nil, nil
}

func (h *OcspHandler) Start() {
	for {
		res, err := h.getResponse()
		h.crt.OCSPStaple = res

		if h.isRevoked {
			break
		}

		var sleep time.Duration
		if err != nil || h.ocspNextUpdate.IsZero() {
			sleep = 5 * time.Minute
		} else {
			sleep = h.ocspNextUpdate.Sub(time.Now())
		}
		time.Sleep(sleep)
	}
}

func NewOcspHandler(crt tls.Certificate) (*OcspHandler, error) {
	if len(crt.Certificate) < 2 {
		return nil, errors.New("no issuer in chain")
	}
	leaf, err := x509.ParseCertificate(crt.Certificate[0])
	if err != nil {
		return nil, err
	}
	issuer, err := x509.ParseCertificate(crt.Certificate[1])
	if err != nil {
		return nil, err
	}

	return &OcspHandler{
		crt:    crt,
		leaf:   leaf,
		issuer: issuer,
		client: &http.Client{Timeout: 5 * time.Second},
	}, nil
}

func (h *OcspHandler) GetCertificate(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return &h.crt, nil
}
