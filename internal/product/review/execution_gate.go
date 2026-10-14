package review

import "context"

type ExecutionGate func(context.Context, ExecutionProvenance) (func(), error)

func (s *Service) WithExecutionGate(gate ExecutionGate) *Service { s.executionGate = gate; return s }

func (s *Service) acquireExecutionGate(ctx context.Context, p *ExecutionProvenance) (func(), error) {
	if p == nil || p.SkillVersionRef.Absent() {
		return func() {}, nil
	}
	if !p.Valid() {
		return nil, ErrInvalid
	}
	if s.executionGate == nil {
		return nil, ErrUnavailable
	}
	release, e := s.executionGate(ctx, *p)
	if e != nil {
		if release != nil {
			release()
		}
		return nil, e
	}
	if release == nil {
		return nil, ErrUnavailable
	}
	return release, nil
}
func sameExecution(a, b *ExecutionProvenance) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
func (s *ExecutionCandidateService) WithExecutionGate(gate ExecutionGate) *ExecutionCandidateService {
	s.service.WithExecutionGate(gate)
	return s
}
func (s *PolicyAdoptionService) WithExecutionGate(gate ExecutionGate) *PolicyAdoptionService {
	s.service.WithExecutionGate(gate)
	return s
}
