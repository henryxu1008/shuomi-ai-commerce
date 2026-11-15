package aiworkbench

import rc "task-processor/internal/reportcenter"

type ReportExecution struct {
	Origin         rc.ScheduledOrigin
	CaptureStarted bool
	ClosedNoWrite  bool
	ReportID       string
	ReportDigest   string
	Proof          string
}
