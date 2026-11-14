package main

import "testing"

func TestProvenanceBindsIndependentRecordsAndLiveIdentities(t *testing.T) {
	s := sourceObservation{Kind: "current-read-only-observation-not-historical-proof", ContainerID: "source", ImageID: "image", Databases: map[string]sourceDatabase{"product_agent": {SystemID: "source-pg", Database: "product_agent", OID: "101", SessionUser: "business_cluster_admin", DumpSHA256: "dump"}}}
	r := isolatedRestore{Kind: "isolated-restore-observation-v1", SourceObservationSHA256: "source-digest", SourceDatabase: "product_agent", DumpSHA256: "dump", ContainerID: "clone", ImageID: "image", Network: "none", SystemID: "clone-pg", Database: "isolated", DatabaseOID: "201", SessionUser: "chat1260_restore_admin", ExitOnError: true}
	p := isolatedPin{ContainerID: "clone", SystemID: "clone-pg"}
	live := liveProvenance{SourceContainer: "source", SourceImage: "image", SourceSystemID: "source-pg", SourceDBOID: "101", TargetContainer: "clone", TargetImage: "image", TargetNetwork: "none", TargetLabel: "chat1260-isolated", TargetSystemID: "clone-pg", TargetDBOID: "201", TargetDatabase: "isolated", TargetSessionUser: "chat1260_restore_admin"}
	if err := validateProvenance(s, r, p, live, "source-digest", "dump"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*sourceObservation, *isolatedRestore, *isolatedPin, *liveProvenance)
	}{
		{"different_source", func(s *sourceObservation, r *isolatedRestore, p *isolatedPin, l *liveProvenance) {
			l.SourceContainer = "other"
		}},
		{"different_target", func(s *sourceObservation, r *isolatedRestore, p *isolatedPin, l *liveProvenance) {
			l.TargetContainer = "other"
		}},
		{"wrong_target_oid", func(s *sourceObservation, r *isolatedRestore, p *isolatedPin, l *liveProvenance) {
			l.TargetDBOID = "202"
		}},
		{"network_changed", func(s *sourceObservation, r *isolatedRestore, p *isolatedPin, l *liveProvenance) {
			l.TargetNetwork = "bridge"
		}},
		{"pin_changed", func(s *sourceObservation, r *isolatedRestore, p *isolatedPin, l *liveProvenance) {
			p.SystemID = "other"
		}},
		{"wrong_receipt", func(s *sourceObservation, r *isolatedRestore, p *isolatedPin, l *liveProvenance) {
			r.SourceObservationSHA256 = "other"
		}},
		{"not_restored", func(s *sourceObservation, r *isolatedRestore, p *isolatedPin, l *liveProvenance) {
			r.ExitOnError = false
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ss, rr, pp, ll := s, r, p, live
			tc.change(&ss, &rr, &pp, &ll)
			if err := validateProvenance(ss, rr, pp, ll, "source-digest", "dump"); err == nil {
				t.Fatal("incorrect provenance accepted")
			}
		})
	}
}
