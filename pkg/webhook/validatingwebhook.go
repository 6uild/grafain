package webhook

import (
	"context"
	"net/http"

	"github.com/alpe/grafain/pkg/artifact"
	"github.com/iov-one/weave/app"
	"github.com/iov-one/weave/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ inject.Client = &podValidator{}
var _ admission.DecoderInjector = &podValidator{}

type queryStore interface {
	Query(abci.RequestQuery) abci.ResponseQuery
}
type podValidator struct {
	logger  log.Logger
	client  client.Client
	decoder *admission.Decoder
	store   queryStore
}

func NewPodValidator(store queryStore, logger log.Logger) *podValidator {
	return &podValidator{
		logger: logger,
		store:  store,
	}
}

// Handle accepts all pod admission requests
func (v *podValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := v.logger.With("uid", req.UID, "kind", req.Kind.Kind, "req", req)
	logger.Debug("Starting pod admission")
	defer func() { logger.Debug("Finished pod admission") }()

	pod := &corev1.Pod{}
	err := v.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	logger.Info("Handling pod", "pod", pod)

	if err := v.doWithContainers(pod.Spec.Containers); err != nil {
		return admission.Errored(encodeErr(err))
	}
	if err := v.doWithContainers(pod.Spec.InitContainers); err != nil {
		return admission.Errored(encodeErr(err))
	}
	return admission.Allowed("")
}

var internalToHttpCode = map[error]int32{
	errors.ErrNotFound:     http.StatusNotFound,
	errors.ErrUnauthorized: http.StatusUnauthorized,
	errors.ErrDuplicate:    http.StatusConflict,
}

func encodeErr(err error) (int32, error) {
	err = errors.Redact(err)
	if c, ok := internalToHttpCode[err]; ok {
		return c, err
	}
	return http.StatusInternalServerError, err
}

const queryPath = "/artifacts/image"

func (v *podValidator) doWithContainers(containers []corev1.Container) error {
	for _, c := range containers {
		v.logger.Info("inspecting container", "image", c.Image, "name", c.Name)
		resp := v.store.Query(abci.RequestQuery{
			Path: queryPath,
			Data: []byte(c.Image),
		})
		v.logger.Debug("query response", "resp", resp)
		if resp.Code != 0 {
			return errors.Wrap(errors.ErrDatabase, resp.Log)
		}
		var vals app.ResultSet
		if err := vals.Unmarshal(resp.Value); err != nil {
			return errors.Wrap(err, "failed to unmarshal client response")
		}
		if len(vals.Results) == 0 {
			return errors.ErrNotFound
		}

		artfs := make([]artifact.Artifact, len(vals.Results))
		for i, v := range vals.Results {
			var artf artifact.Artifact
			if err := artf.Unmarshal(v); err != nil {
				return errors.Wrapf(err, "failed to unmarshal client response")
			}
			artfs[i] = artf
		}
		// further checks
	}
	return nil
}

func (v *podValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

func (v *podValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
