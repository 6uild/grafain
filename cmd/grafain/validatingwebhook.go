package main

import (
	"context"
	"net/http"

	"github.com/tendermint/tendermint/libs/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ inject.Client = &podValidator{}
var _ admission.DecoderInjector = &podValidator{}

type podValidator struct {
	logger  log.Logger
	client  client.Client
	decoder *admission.Decoder
}

// Handle accepts all pod admission requests
func (v *podValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := v.logger.With("uid", req.UID, "kind", req.Kind.Kind, "req", req)
	logger.Info("starting pod admission")
	defer func() { logger.Info("finished pod admission") }()

	pod := &corev1.Pod{}

	err := v.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	logger.Info("handling pod", "pod", pod)

	if err := v.doWithContainers(pod.Spec.Containers); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if err := v.doWithContainers(pod.Spec.InitContainers); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.Allowed("grafain noop default")
}

func (v *podValidator) doWithContainers(containers []corev1.Container) error {
	for _, c := range containers {
		v.logger.Info("inspecting container", "image", c.Image, "name", c.Name)
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
