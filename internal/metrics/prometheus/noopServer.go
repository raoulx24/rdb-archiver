package prometheus

import "context"

type noopServer struct{}

func (noopServer) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
