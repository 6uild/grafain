package xadmission

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	v1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

const JSONContentType = "application/json"

// Define constants for metav1.Status.Status
// See https://github.com/kubernetes/kubernetes/blob/release-1.1/docs/devel/api-conventions.md#response-status-kind

const (
	successMessage = "Successfully admitted."
)

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

func ReviewHandler(w http.ResponseWriter, r *http.Request, logger log.Logger) {
	level.Debug(logger).Log("message", "Starting admission review handler")
	var admitResponse v1.AdmissionReview
	ar, err := deserializeRequest(r)
	if err != nil {
		admitResponse.Response = &v1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Status:  metav1.StatusSuccess,
				Message: err.Error(),
			},
		}
		if err := RespondJson(w, http.StatusBadRequest, admitResponse); err != nil {
			level.Error(logger).Log("message", "failed to write response", "cause", err)
		}
		return
	}
	admitResponse.Response = &v1.AdmissionResponse{
		UID:     ar.Request.UID,
		Allowed: true,
		Result: &metav1.Status{
			Status:  metav1.StatusSuccess,
			Message: successMessage,
		},
	}

	//for k8sType, handler := range handlers {
	//	if ar.Request.Kind.Kind == k8sType {
	//		if err := handler(&ar, admitResponse, config); err != nil {
	//			glog.Errorf("handler failed: %v", err)
	//			http.Error(w, "Whoops! The handler failed!", http.StatusInternalServerError)
	//			return
	//		}
	//
	//	}
	//}

	// Send response
	if err := RespondJson(w, http.StatusOK, admitResponse); err != nil {
		level.Error(logger).Log("message", "failed to write response", "cause", err)
	}
	return
}

func deserializeRequest(r *http.Request) (ar v1.AdmissionReview, err error) {
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		return ar, fmt.Errorf("cannot to read body")
	}

	deserializer := codecs.UniversalDeserializer()
	_, _, err = deserializer.Decode(body, nil, &ar)
	if err != nil {
		return ar, fmt.Errorf("failed to marshal %v", err)
	}
	if ar.Request == nil {
		return ar, fmt.Errorf("admission request is empty")
	}
	return ar, nil
}
