package browser

import (
	"math"
	"strings"
	"sync"
	"time"
)

type ProxyPoolConfig struct {
	Enabled                bool
	Strategy               string
	FailureCooldownSeconds int
	Proxies                []string
}

type proxyAssignment struct {
	server        string
	assignments   int64
	successes     int64
	failures      int64
	cooldowns     int64
	lastSuccessAt time.Time
	lastFailureAt time.Time
	cooldownUntil time.Time
}

type ProxyPool struct {
	enabled     bool
	strategy    string
	cooldown    time.Duration
	assignments []*proxyAssignment
	nextIndex   int
	mu          sync.Mutex
}

func NewProxyPool(cfg ProxyPoolConfig) *ProxyPool {
	if !cfg.Enabled || len(cfg.Proxies) == 0 {
		return nil
	}

	assignments := make([]*proxyAssignment, 0, len(cfg.Proxies))
	for _, proxy := range cfg.Proxies {
		proxy = strings.TrimSpace(proxy)
		if proxy == "" {
			continue
		}
		assignments = append(assignments, &proxyAssignment{server: proxy})
	}
	if len(assignments) == 0 {
		return nil
	}

	return &ProxyPool{
		enabled:     true,
		strategy:    strings.ToLower(strings.TrimSpace(cfg.Strategy)),
		cooldown:    time.Duration(cfg.FailureCooldownSeconds) * time.Second,
		assignments: assignments,
	}
}

func (pp *ProxyPool) Acquire() string {
	if pp == nil || !pp.enabled || len(pp.assignments) == 0 {
		return ""
	}

	pp.mu.Lock()
	defer pp.mu.Unlock()

	if pp.strategy == "" {
		pp.strategy = "round_robin"
	}

	now := time.Now()
	if pp.strategy == "health_aware" {
		if assignment := pp.selectHealthiestAvailable(now); assignment != nil {
			assignment.assignments++
			return assignment.server
		}
	}

	total := len(pp.assignments)
	for i := 0; i < total; i++ {
		index := (pp.nextIndex + i) % total
		assignment := pp.assignments[index]
		if assignment == nil || assignment.server == "" {
			continue
		}
		if assignment.cooldownUntil.After(now) {
			continue
		}
		assignment.assignments++
		pp.nextIndex = (index + 1) % total
		return assignment.server
	}

	index := pp.nextIndex % total
	pp.nextIndex = (index + 1) % total
	assignment := pp.assignments[index]
	if assignment == nil {
		return ""
	}
	assignment.assignments++
	return assignment.server
}

func (pp *ProxyPool) MarkSuccess(proxyServer string) {
	if pp == nil || proxyServer == "" {
		return
	}

	pp.mu.Lock()
	defer pp.mu.Unlock()

	for _, assignment := range pp.assignments {
		if assignment == nil || assignment.server != proxyServer {
			continue
		}
		assignment.successes++
		assignment.lastSuccessAt = time.Now()
		return
	}
}

func (pp *ProxyPool) MarkFailure(proxyServer string) {
	if pp == nil || proxyServer == "" {
		return
	}

	pp.mu.Lock()
	defer pp.mu.Unlock()

	for _, assignment := range pp.assignments {
		if assignment == nil || assignment.server != proxyServer {
			continue
		}
		assignment.failures++
		assignment.lastFailureAt = time.Now()
		assignment.cooldownUntil = time.Now().Add(pp.cooldown)
		assignment.cooldowns++
		return
	}
}

func (pp *ProxyPool) selectHealthiestAvailable(now time.Time) *proxyAssignment {
	var selected *proxyAssignment
	bestScore := math.Inf(-1)

	for _, assignment := range pp.assignments {
		if assignment == nil || assignment.server == "" {
			continue
		}
		if assignment.cooldownUntil.After(now) {
			continue
		}

		score := calculateProxyHealthScore(assignment, now)
		if selected == nil || score > bestScore {
			selected = assignment
			bestScore = score
		}
	}

	return selected
}

func calculateProxyHealthScore(assignment *proxyAssignment, now time.Time) float64 {
	if assignment == nil {
		return math.Inf(-1)
	}

	score := 100.0
	score += float64(assignment.successes) * 8
	score -= float64(assignment.failures) * 15
	score -= float64(assignment.cooldowns) * 10
	score -= float64(assignment.assignments)

	if !assignment.lastSuccessAt.IsZero() {
		successAgeMinutes := now.Sub(assignment.lastSuccessAt).Minutes()
		score += math.Max(0, 20-successAgeMinutes)
	}
	if !assignment.lastFailureAt.IsZero() {
		failureAgeMinutes := now.Sub(assignment.lastFailureAt).Minutes()
		score -= math.Max(0, 30-failureAgeMinutes)
	}

	return score
}
