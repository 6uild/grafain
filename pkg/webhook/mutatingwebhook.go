package webhook

import (
	"context"

	grafain "github.com/alpe/grafain/pkg/client"
	"github.com/cruise-automation/k-rail/policies"
	"github.com/cruise-automation/k-rail/resource"
	"github.com/cruise-automation/k-rail/server"
	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/log"
	"gomodules.xyz/jsonpatch/v2"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var _ inject.Client = &mutatingWebHook{}
var _ admission.DecoderInjector = &mutatingWebHook{}

type queryStore interface {
	AbciQuery(path string, data []byte) (grafain.AbciResponse, error)
}
type mutatingWebHook struct {
	logger log.Logger
	client client.Client

	source queryStore // artifact source

	Config             server.Config
	EnforcedPolicies   []server.Policy
	ReportOnlyPolicies []server.Policy
	Exemptions         []policies.CompiledExemption
}

func NewMutatingWebHook(source queryStore, logger log.Logger) (*mutatingWebHook, error) {
	exemp :=
		`
- resource_name: "*"
  namespace: "kube-system"
  username: "*"
  group: "*"
  exempt_policies: ["*"]
`
	exemptions, err := policies.ExemptionsFromYAML([]byte(exemp))
	if err != nil {
		return nil, errors.Wrap(err, "setup exemptions")
	}
	r := &mutatingWebHook{
		logger:     logger,
		source:     source,
		Exemptions: exemptions,
		Config: server.Config{
			Policies: []server.PolicySettings{
				{
					Name:       "pod_empty_dir_size_limit",
					Enabled:    true,
					ReportOnly: false,
				},
			},
			PolicyConfig: policies.Config{
				PolicyTrustedRepositoryRegexes: []string{
					"^k8s.gcr.io/.*",     // official k8s GCR repo
					`^[A-Za-z0-9\-:@]+$`, // official docker hub images
				},
				MutateEmptyDirSizeLimit: policies.MutateEmptyDirSizeLimit{
					MaximumSizeLimit: apiresource.MustParse("1Gi"),
					DefaultSizeLimit: apiresource.MustParse("512Mi"),
				},
			}},
	}
	registerDefaultPolicies(r)
	//r.EnforcedPolicies = append(r.EnforcedPolicies, policy.NewArtifactWhitelist(source))
	return r, nil
}

// Handle all admission requests
func (s *mutatingWebHook) Handle(ctx context.Context, req admission.Request) admission.Response {
	var enforcedViolations []policies.ResourceViolation
	var reportedViolations []policies.ResourceViolation
	var exemptViolations []policies.ResourceViolation
	var mutationPatches []policies.PatchOperation

	// allow resource if namespace is blacklisted (= not handled by us)
	for _, namespace := range s.Config.BlacklistedNamespaces {
		if namespace == req.Namespace {
			return admission.Allowed("admission review - blacklisted namespace")
		}
	}

	for _, val := range s.EnforcedPolicies {
		violations, patches := val.Validate(ctx, s.Config.PolicyConfig, &req.AdmissionRequest)

		// render non-exempt Pod mutations
		// TODO: This could use a bit of refactoring so there is less repetition and we could
		// have the relevant resource name available for any resource being checked for exemptions.
		// The AdmissionReview Name is often empty and populated by an downstream controller.
		podResource := resource.GetPodResource(&req.AdmissionRequest)
		if len(violations) == 0 && patches != nil && !policies.IsExempt(
			podResource.ResourceName,
			req.Namespace,
			req.UserInfo,
			val.Name(),
			s.Exemptions,
		) {
			mutationPatches = append(mutationPatches, patches...)
		}

		// apply exempt and non-exempt violations
		if len(violations) > 0 {
			if policies.IsExempt(
				violations[0].ResourceName,
				req.Namespace,
				req.UserInfo,
				val.Name(),
				s.Exemptions,
			) {
				exemptViolations = append(exemptViolations,
					violations...)
			} else {
				enforcedViolations = append(enforcedViolations,
					violations...)
			}
		}
	}

	for _, val := range s.ReportOnlyPolicies {
		violations, _ := val.Validate(ctx, s.Config.PolicyConfig, &req.AdmissionRequest)
		if len(violations) > 0 {
			if policies.IsExempt(
				violations[0].ResourceName,
				req.Namespace,
				req.UserInfo,
				val.Name(),
				s.Exemptions,
			) {
				exemptViolations = append(exemptViolations,
					violations...)
			} else {
				reportedViolations = append(reportedViolations,
					violations...)
			}
		}
	}

	s.printAuditLogs(req.UserInfo.Username, exemptViolations, reportedViolations, enforcedViolations)

	if len(enforcedViolations) > 0 && s.Config.GlobalReportOnly == false {
		violations := ""
		for _, v := range enforcedViolations {
			violations = violations + "\n" + v.HumanString()
		}
		return admission.Denied(violations)
	}

	// allow other resources, but include any reported violations
	var violations string
	for _, v := range reportedViolations {
		violations = violations + "\n" + v.HumanString()
	}
	if len(violations) != 0 {
		violations = "NOT ENFORCED:\n" + violations
	} else {
		violations = "NO VIOLATIONS"
	}

	// todo: revisit type conversion: both go into json anyway
	patches := make([]jsonpatch.JsonPatchOperation, len(mutationPatches))
	for i, v := range mutationPatches {
		patches[i] = jsonpatch.JsonPatchOperation{
			Operation: v.Op,
			Path:      v.Path,
			Value:     v.Value,
		}
	}
	return admission.Patched(violations, patches...)
}

func (s *mutatingWebHook) printAuditLogs(username string, exemptViolations []policies.ResourceViolation, reportedViolations []policies.ResourceViolation, enforcedViolations []policies.ResourceViolation) {
	for _, v := range exemptViolations {
		s.logger.Info("EXEMPT",
			"kind", v.ResourceKind,
			"resource", v.ResourceName,
			"namespace", v.Namespace,
			"policy", v.Policy,
			"user", username,
			"enforced", false,
		)
	}

	// log report-only violations
	for _, v := range reportedViolations {
		s.logger.Info("NOT ENFORCED",
			"kind", v.ResourceKind,
			"resource", v.ResourceName,
			"namespace", v.Namespace,
			"policy", v.Policy,
			"user", username,
			"enforced", false,
		)
	}

	// log enforced violations when in global report-only mode
	if s.Config.GlobalReportOnly {
		for _, v := range enforcedViolations {
			s.logger.Info("NOT ENFORCED",
				"kind", v.ResourceKind,
				"resource", v.ResourceName,
				"namespace", v.Namespace,
				"policy", v.Policy,
				"user", username,
				"enforced", false,
			)
		}
	}

	// log and respond to enforced violations
	if len(enforcedViolations) > 0 && s.Config.GlobalReportOnly == false {
		for _, v := range enforcedViolations {
			s.logger.Info("ENFORCED",
				"kind", v.ResourceKind,
				"resource", v.ResourceName,
				"namespace", v.Namespace,
				"policy", v.Policy,
				"user", username,
				"enforced", true,
			)
		}
	}
}

//var internalToHttpCode = map[error]int32{
//	errors.ErrNotFound:     http.StatusNotFound,
//	errors.ErrUnauthorized: http.StatusUnauthorized,
//	errors.ErrDuplicate:    http.StatusConflict,
//}
//
//func encodeErr(err error) (int32, error) {
//	err = errors.Redact(err)
//	if c, ok := internalToHttpCode[err]; ok {
//		return c, err
//	}
//	return http.StatusInternalServerError, err
//}

func (s *mutatingWebHook) InjectClient(c client.Client) error {
	s.client = c
	return nil
}

func (s *mutatingWebHook) InjectDecoder(d *admission.Decoder) error {
	//v.decoder = d
	return nil
}
