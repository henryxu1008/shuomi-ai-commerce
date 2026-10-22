package main

import (
	"encoding/json"
	"flag"
	"os"
	"task-processor/internal/app/runtime/sheinloginworker"
	"task-processor/internal/pkg/appenv"
	"task-processor/internal/sheinlogin"
)

func main() {
	var o sheinloginworker.Options
	flag.StringVar(&o.ConfigPath, "config", "", "absolute private current worker JSON config")
	flag.StringVar(&o.Action, "action", "serve", "serve|login|status|verify|cancel|activate|quiesce")
	flag.StringVar(&o.OrganizationID, "organization", "", "current opaque Organization ID")
	flag.StringVar(&o.StoreID, "store", "", "current Store UUID")
	flag.StringVar(&o.TokenFile, "token-file", "", "absolute private current caller JWT file")
	flag.StringVar(&o.RequestKey, "request-key", "", "original request UUID for login/activate/quiesce; reuse after UNKNOWN")
	flag.StringVar(&o.AttemptID, "attempt", "", "original attempt UUID for status/verify/cancel")
	flag.BoolVar(&o.ForceLogin, "force-login", false, "explicit fresh login; never implied by status")
	flag.Int64Var(&o.ExpectedGeneration, "expected-generation", 0, "confirmed closed prior generation for activation CAS")
	flag.IntVar(&o.Port, "health-port", 8086, "serve-only health port; 0 disables")
	level := flag.String("log-level", "info", "log level")
	flag.Parse()
	logger := appenv.SetupLoggerWithLevel(*level)
	if flag.NArg() != 0 {
		_ = json.NewEncoder(os.Stdout).Encode(sheinlogin.Result{Status: "failed", ErrorCode: "NOT_ALLOWED_OR_INVALID"})
		os.Exit(1)
	}
	if err := sheinloginworker.Run(logger, o); err != nil {
		_ = json.NewEncoder(os.Stdout).Encode(sheinlogin.Result{Status: "failed", ErrorCode: sheinlogin.PublicErrorCode(err)})
		os.Exit(1)
	}
}
