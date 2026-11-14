package main

import (
	"errors"
	"time"
)

// The inputs must originate from separately protected source observation,
// restore receipt, previous isolated pin, and a fresh Docker/DB observation.
// A caller-supplied manifest is never by itself sufficient to admit a write.
type sourceDatabase struct{ SystemID, Database, OID, SessionUser, DumpSHA256 string }
type sourceObservation struct {
	Kind, ContainerID, ImageID string
	Databases                  map[string]sourceDatabase
}
type isolatedRestore struct {
	Kind, SourceObservationSHA256, SourceDatabase, DumpSHA256, ContainerID, ImageID, Network, SystemID, Database, DatabaseOID, SessionUser string
	ExitOnError                                                                                                                            bool
	RestoredAt                                                                                                                             time.Time
}
type isolatedPin struct{ ContainerID, SystemID string }
type liveProvenance struct{ SourceContainer, SourceImage, SourceSystemID, SourceDBOID, TargetContainer, TargetImage, TargetNetwork, TargetLabel, TargetSystemID, TargetDBOID, TargetDatabase, TargetSessionUser string }

func validateProvenance(s sourceObservation, r isolatedRestore, p isolatedPin, l liveProvenance, sourceFileSHA, dumpSHA string) error {
	d, ok := s.Databases["product_agent"]
	if !ok || s.Kind != "current-read-only-observation-not-historical-proof" || r.Kind != "isolated-restore-observation-v1" || r.SourceDatabase != "product_agent" || d.Database != "product_agent" || d.SessionUser != "business_cluster_admin" || !r.ExitOnError || sourceFileSHA == "" || r.SourceObservationSHA256 != sourceFileSHA || dumpSHA == "" || d.DumpSHA256 != dumpSHA || r.DumpSHA256 != dumpSHA || s.ContainerID == "" || s.ContainerID != l.SourceContainer || s.ImageID == "" || s.ImageID != l.SourceImage || d.SystemID == "" || d.SystemID != l.SourceSystemID || d.OID == "" || d.OID != l.SourceDBOID || r.ContainerID == "" || r.ContainerID != p.ContainerID || r.ContainerID != l.TargetContainer || r.ImageID != s.ImageID || r.ImageID != l.TargetImage || r.Network != "none" || l.TargetNetwork != "none" || l.TargetLabel != "chat1260-isolated" || r.SystemID == "" || r.SystemID == d.SystemID || r.SystemID != p.SystemID || r.SystemID != l.TargetSystemID || r.Database == "" || r.Database != l.TargetDatabase || r.DatabaseOID == "" || r.DatabaseOID != l.TargetDBOID || r.SessionUser != "chat1260_restore_admin" || r.SessionUser != l.TargetSessionUser {
		return errors.New("source, isolated restore and live identity do not match")
	}
	return nil
}
