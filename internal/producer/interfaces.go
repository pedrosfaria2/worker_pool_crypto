package producer

import (
	"context"
)

type Producer interface {
	Connect(ctx context.Context) error
	Start(ctx context.Context) error
	Stop() error
}
