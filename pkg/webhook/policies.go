package webhook

import (
	"context"

	"github.com/cruise-automation/k-rail/policies"
	"github.com/cruise-automation/k-rail/policies/ingress"
	"github.com/cruise-automation/k-rail/policies/pod"
	log "github.com/sirupsen/logrus"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
)

// Policy specifies how a Policy is implemented
// Returns a slice of violations and an optional slice of patch operations if mutation is desired.
type Policy interface {
	Name() string
	Validate(ctx context.Context,
		config policies.Config,
		ar *admissionv1beta1.AdmissionRequest,
	) ([]policies.ResourceViolation, []policies.PatchOperation)
}

func registerDefaultPolicies(s *mutatingWebHook) {
	// Policies will be run in the order that they are registered.
	// Policies that mutate will have their resulting patch merged with any previous patches in that order as well.

	registerPolicy(s, pod.PolicyNoExec{})
	registerPolicy(s, pod.PolicyBindMounts{})
	registerPolicy(s, pod.PolicyDockerSock{})
	registerPolicy(s, pod.PolicyImageImmutableReference{})
	registerPolicy(s, pod.PolicyNoTiller{})
	registerPolicy(s, pod.PolicyTrustedRepository{})
	registerPolicy(s, pod.PolicyNoHostNetwork{})
	registerPolicy(s, pod.PolicyNoPrivilegedContainer{})
	registerPolicy(s, pod.PolicyNoNewCapabilities{})
	registerPolicy(s, pod.PolicyNoHostPID{})
	registerPolicy(s, pod.PolicySafeToEvict{})
	registerPolicy(s, pod.PolicyMutateSafeToEvict{})
	registerPolicy(s, pod.PolicyDefaultSeccompPolicy{})
	registerPolicy(s, pod.PolicyNoShareProcessNamespace{})
	registerPolicy(s, ingress.PolicyRequireIngressExemption{})
}

func registerPolicy(s *mutatingWebHook, v Policy) {
	found := false
	for _, val := range s.Config.Policies {
		if val.Name == v.Name() {
			found = true
			if val.Enabled {
				if s.Config.GlobalReportOnly {
					s.ReportOnlyPolicies = append(s.ReportOnlyPolicies, v)
					log.Infof("enabling %s validator in REPORT ONLY mode because GLOBAL REPORT ONLY MODE is on", v.Name())
				} else if val.ReportOnly {
					s.ReportOnlyPolicies = append(s.ReportOnlyPolicies, v)
					log.Infof("enabling %s validator in REPORT ONLY mode", v.Name())
				} else {
					s.EnforcedPolicies = append(s.EnforcedPolicies, v)
					log.Infof("enabling %s validator in ENFORCE mode", v.Name())
				}
			} else {
				log.Infof("validator %s is NOT ENABLED", v.Name())

			}
		}
	}
	if !found {
		s.ReportOnlyPolicies = append(s.ReportOnlyPolicies, v)
		log.Warnf("configuration not present for %s validator, enabling REPORT ONLY mode", v.Name())
	}
}
