package xadmission

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"k8s.io/api/admission/v1beta1"
)

func TestHandleReview(t *testing.T) {
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ReviewHandler(w, r, logger)
	}))
	defer s.Close()

	specs := map[string]struct {
		src     *v1beta1.AdmissionRequest
		expCode int
	}{
		"Query admission control with valid request": {
			src:     &v1beta1.AdmissionRequest{},
			expCode: http.StatusOK,
		},
		"Query admission control with invalid request": {
			src:     nil,
			expCode: http.StatusBadRequest,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			ar := v1beta1.AdmissionReview{
				Request: spec.src,
			}
			blob, err := json.Marshal(ar)
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			resp, err := http.Post(s.URL, "", bytes.NewReader(blob))
			if err != nil {
				t.Fatalf("unexpected error: %+v", err)
			}
			if exp, got := spec.expCode, resp.StatusCode; exp != got {
				t.Errorf("expected %v but got %v", exp, got)
			}
		})
	}
}
