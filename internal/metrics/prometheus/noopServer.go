package prometheus

import "context"

type noopServer struct{}

func (noopServer) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (noopServer) Stop(ctx context.Context) error {
	return nil
}

func (noopServer) NeedsRestart(newCfg Config) bool {
	return newCfg.Enabled
}

func (noopServer) NeedMetricsRebuild(newCfg Config) bool {
	return false
}

func (noopServer) UpdateConfig(cfg Config) {
}
