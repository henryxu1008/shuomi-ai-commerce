package projectcenter

import (
	"context"
	"sort"
	"task-processor/internal/authidentity"
	"time"
	_ "time/tzdata"
)

func (s *Service) Overview(ctx context.Context, scope Scope, mode, zone string) (Overview, error) {
	return s.overviewAt(ctx, scope, mode, zone, time.Now())
}
func (s *Service) overviewAt(ctx context.Context, scope Scope, mode, zone string, now time.Time) (Overview, error) {
	if e := s.authorized(ctx, scope, false); e != nil {
		return Overview{}, e
	}
	if mode != "active" && mode != "recent" && mode != "archived" || len(zone) > 128 || zone == "Local" {
		return Overview{}, ErrInvalid
	}
	if zone == "" {
		zone = "UTC"
	}
	loc, e := time.LoadLocation(zone)
	if e != nil {
		return Overview{}, ErrInvalid
	}
	local := now.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	snapshot, e := s.Store.Overview(ctx, scope, mode, start, start.AddDate(0, 0, 1))
	if e != nil {
		return Overview{}, e
	}
	if snapshot.Scope != scope || snapshot.ActiveProjects < 0 || snapshot.TodayUpdatedProjects < 0 || snapshot.TodayUpdatedProjects > snapshot.ActiveProjects || snapshot.StoreProjects < 0 || snapshot.GeneralProjects < 0 || snapshot.StoreProjects+snapshot.GeneralProjects != snapshot.ActiveProjects || snapshot.ActiveProjects > 9007199254740991 || len(snapshot.TaskIDs) > 201 || len(snapshot.StoreIDs) > 101 {
		return Overview{}, ErrUnavailable
	}
	out := Overview{ActiveProjects: snapshot.ActiveProjects, TodayUpdatedProjects: snapshot.TodayUpdatedProjects, StoreProjects: snapshot.StoreProjects, GeneralProjects: snapshot.GeneralProjects, TimeZone: zone, Today: start.Format("2006-01-02"), Stores: []StoreOption{}, StoresAvailable: true}
	taskCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	waiting, complete := 0, len(snapshot.TaskIDs) <= 200
	seen := map[string]bool{}
	if complete {
		for _, id := range snapshot.TaskIDs {
			if !ValidID(id) {
				cancel()
				return Overview{}, ErrUnavailable
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			if taskCtx.Err() != nil {
				complete = false
				break
			}
			v := s.resolve(taskCtx, scope, Reference{Kind: "BUSINESS_TASK", TargetID: id})
			if !v.Available || v.TaskState == "" {
				complete = false
				break
			}
			if v.TaskState == "WAITING_CONFIRMATION" {
				waiting++
			}
		}
	}
	cancel()
	if complete {
		out.WaitingTasks = &waiting
	}
	storeCtx, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	if len(snapshot.StoreIDs) > 100 {
		out.StoresAvailable = false
	} else {
		seen = map[string]bool{}
		for _, id := range snapshot.StoreIDs {
			if !authidentity.IsBoundedIdentifier(id) {
				return Overview{}, ErrUnavailable
			}
			if seen[id] {
				continue
			}
			seen[id] = true
			if storeCtx.Err() != nil {
				out.StoresAvailable = false
				break
			}
			v := s.resolve(storeCtx, scope, Reference{Kind: "STORE", TargetID: id})
			if !v.Available {
				out.StoresAvailable = false
				continue
			}
			out.Stores = append(out.Stores, StoreOption{ID: id, Name: v.Title})
		}
	}
	sort.Slice(out.Stores, func(i, j int) bool {
		if out.Stores[i].Name == out.Stores[j].Name {
			return out.Stores[i].ID < out.Stores[j].ID
		}
		return out.Stores[i].Name < out.Stores[j].Name
	})
	if e := s.authorized(ctx, scope, false); e != nil {
		return Overview{}, e
	}
	return out, nil
}
