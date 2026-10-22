package bootstrap

import (
	"context"
	"github.com/sirupsen/logrus"
	"task-processor/internal/sheinlogin"
	"time"
)

type BuildInput struct {
	Logs   func(string) *logrus.Entry
	Config sheinlogin.Config
	Owner  sheinlogin.StoreOwner
}
type BuildResult struct {
	Service *sheinlogin.Service
	Close   func() error
}

func BuildService(input BuildInput) (*BuildResult, error) {
	if input.Config.Validate() != nil || input.Owner == nil {
		return nil, sheinlogin.ErrInvalidInput
	}
	redis, err := sheinlogin.NewRedisStore(input.Config.Redis, input.Config.InstallationID)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redis.Initialize(ctx); err != nil {
		redis.Close()
		return nil, err
	}
	service, err := sheinlogin.NewService(input.Config, redis, input.Owner, input.Logs)
	if err != nil {
		redis.Close()
		return nil, err
	}
	return &BuildResult{Service: service, Close: service.Close}, nil
}
