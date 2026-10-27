package aicapability

type RoutingMode string

const RoutingModeActive RoutingMode = "active"

type CacheStatus string

const (
	CacheStatusNotApplicable CacheStatus = "not_applicable"
	CacheStatusHit           CacheStatus = "hit"
	CacheStatusMiss          CacheStatus = "miss"
)
