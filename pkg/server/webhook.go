package server

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"io/ioutil"
	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"net/http"
)

const JSONContentType = "application/json"

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()

	// (https://github.com/kubernetes/kubernetes/issues/57982)
	//defaulter = runtime.ObjectDefaulter(runtimeScheme)
)

//var ignoredNamespaces = []string {
//	metav1.NamespaceSystem,
//	metav1.NamespacePublic,
//}

type WebhookServer struct {
	logger        log.Logger
}

func NewWebhookServer(l log.Logger) *WebhookServer {
	return &WebhookServer{logger: l}
}


func (s *WebhookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if JSONContentType != r.Header.Get("Content-Type") {
		http.Error(w, "Invalid Content-Type, expect `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	if len(body) == 0 {
		http.Error(w, "Empty body", http.StatusBadRequest)
		return
	}

	var ar v1beta1.AdmissionReview
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		level.Debug(s.logger).Log("message", "Can not deserialize body", "cause", err)
		sendErrorResponse(w, err.Error()) // revisit: do not expose internals
		return
	}

	if ar.Request == nil {
		sendErrorResponse(w, "Request object must not be empty")
	}

	resp := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID:     ar.Request.UID,
			Allowed: true,
		},
	}
	RespondJson(w, http.StatusBadRequest, resp)
}

// sendErrorResponse sends a proper json object response containing the error message.
func sendErrorResponse(w http.ResponseWriter, msg string) {
	resp := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			Result: &metav1.Status{
				Message: msg,
			},
		},
	}
	RespondJson(w, http.StatusOK, resp)
}

