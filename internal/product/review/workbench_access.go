package review

import "context"

func (s *Service) WithWorkbenchAccess(check func(context.Context, ExecutionProvenance) error) *Service {
	s.workbenchAccess = check
	return s
}

func (s *Service) authorizeWorkbench(ctx context.Context, execution *ExecutionProvenance) error {
	if execution == nil || execution.WorkbenchContextRef.Absent() {
		return nil
	}
	if s.workbenchAccess == nil || !execution.Valid() {
		return ErrUnavailable
	}
	return s.workbenchAccess(ctx, *execution)
}
