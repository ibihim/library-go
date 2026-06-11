package health

import (
	"context"
	"time"

	kmsservice "k8s.io/kms/pkg/service"
)

// healthzOK is the value the KMS plugin returns when healthy.
// See https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kms/apis/v2/api.proto#L39
const healthzOK = "ok"

const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
	statusError     = "error"
)

type PluginHealthReport struct {
	// KeyID is the controller's sequential key id; KEKID is the KMS provider's
	// encryption key id. Distinct identifiers, easy to confuse.
	KeyID       string    `json:"keyID"`
	KEKID       string    `json:"kekID,omitempty"`
	Status      string    `json:"status"`
	LastChecked time.Time `json:"lastChecked"`
	Detail      string    `json:"detail,omitempty"`
}

type plugin struct {
	keyID   string
	service kmsservice.Service
}

type checker struct {
	plugins []plugin
	now     func() time.Time
}

func newChecker(plugins []plugin) *checker {
	return &checker{
		plugins: plugins,
		now:     time.Now,
	}
}

// checkStatus never returns an error: a failed probe is encoded as a report
// with Status "error" so the caller always gets one entry per plugin.
func (c *checker) checkStatus(ctx context.Context) []PluginHealthReport {
	reports := make([]PluginHealthReport, 0, len(c.plugins))

	// We could fan out if performance is an issue.
	for _, p := range c.plugins {
		report := PluginHealthReport{
			KeyID:       p.keyID,
			LastChecked: c.now(),
		}

		resp, err := p.service.Status(ctx)
		switch {
		case err != nil:
			report.Status = statusError
			report.Detail = err.Error()
		case resp.Healthz == healthzOK:
			report.Status = statusHealthy
			report.KEKID = resp.KeyID
		default:
			report.Status = statusUnhealthy
			report.Detail = resp.Healthz
		}

		reports = append(reports, report)
	}
	return reports
}
