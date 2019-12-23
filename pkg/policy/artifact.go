package policy

import (
	"context"

	grafain "github.com/alpe/grafain/pkg/client"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"

	"github.com/cruise-automation/k-rail/policies"
	"github.com/cruise-automation/k-rail/resource"
)

type queryStore interface {
	AbciQuery(path string, data []byte) (grafain.AbciResponse, error)
}

type ArtifactWhitelist struct {
	source queryStore
}

func NewArtifactWhitelist(source queryStore) *ArtifactWhitelist {
	return &ArtifactWhitelist{source: source}
}

func (p ArtifactWhitelist) Name() string {
	// more than pod: job, daemonset ?
	return "pod_artifact_whitelist"
}

func (p ArtifactWhitelist) Validate(ctx context.Context, config policies.Config, ar *admissionv1beta1.AdmissionRequest) ([]policies.ResourceViolation, []policies.PatchOperation) {
	podResource := resource.GetPodResource(ar)
	if podResource == nil {
		return nil, nil
	}

	var resourceViolations []policies.ResourceViolation

	validateContainer := func(container corev1.Container) {
		const violationText = "Docker: Artifact not in whitelist"
		const queryByImage = "/artifacts"
		resp, err := p.source.AbciQuery(queryByImage, []byte(container.Image))
		if err == nil && len(resp.Models) != 0 {
			return
		}
		resourceViolations = append(resourceViolations, policies.ResourceViolation{
			Namespace:    ar.Namespace,
			ResourceName: podResource.ResourceName,
			ResourceKind: podResource.ResourceKind,
			Violation:    violationText,
			Policy:       p.Name(),
			Error:        err,
		})
	}
	for _, container := range podResource.PodSpec.Containers {
		validateContainer(container)
	}
	for _, container := range podResource.PodSpec.InitContainers {
		validateContainer(container)
	}
	return resourceViolations, nil
}
