package testsupport

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/iov-one/weave/weavetest/assert"
)

type AdmissionResponse struct {
	Response struct {
		Allowed bool `json:"allowed"`
		Status  struct {
			Code int `json:"code"`
		}
	} `json:"response"`
}

type admissionClient struct {
	t       *testing.T
	c       *http.Client
	address string
}

func NewAdmissionClient(t *testing.T, certDir, hookAddress, admissionPath string) *admissionClient {
	t.Helper()
	certPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(filepath.Join(certDir, "ca.pem"))
	assert.Nil(t, err)
	certPool.AppendCertsFromPEM(pem)
	return &admissionClient{
		c: &http.Client{
			Timeout: 550 * time.Millisecond,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					ServerName: "grafain.default.svc",
					RootCAs:    certPool,
				},
			},
		},
		t:       t,
		address: fmt.Sprintf("https://%s%s", hookAddress, admissionPath),
	}
}

func (h admissionClient) Query(content string) AdmissionResponse {
	h.t.Helper()
	r, err := h.c.Post(
		h.address,
		"application/json",
		strings.NewReader(content),
	)
	assert.Nil(h.t, err)
	assert.Equal(h.t, http.StatusOK, r.StatusCode)

	codec := json.NewDecoder(io.TeeReader(r.Body, os.Stdout))
	var data AdmissionResponse
	assert.Nil(h.t, codec.Decode(&data))
	return data
}
