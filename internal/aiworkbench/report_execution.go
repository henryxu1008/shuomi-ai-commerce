package aiworkbench

import rc "github.com/henryxu1008/shuomi-ai-commerce/internal/reportcenter"

type ReportExecution struct {
	Origin         rc.ScheduledOrigin
	CaptureStarted bool
	ClosedNoWrite  bool
	ReportID       string
	ReportDigest   string
	Proof          string
}
