package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"task-processor/internal/skill"
)

type officialSkillsPreparer interface{ PrepareOfficial(context.Context) error }

// Failure leaves the existing Skill service and all other services available.
func prepareOfficialSkills(ctx context.Context, preparer officialSkillsPreparer, logger *logrus.Logger) {
	bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := preparer.PrepareOfficial(bounded); err != nil {
		category := "unavailable"
		if errors.Is(err, skill.ErrConflict) {
			category = "conflict"
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(bounded.Err(), context.DeadlineExceeded) {
			category = "timeout"
		}
		logger.WithFields(logrus.Fields{"component": "official-skills", "reason": category}).Warn("official skill preparation failed; existing skills remain available")
	}
}
