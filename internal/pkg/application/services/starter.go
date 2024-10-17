package services

import "context"

type Starter interface {
	Start(ctx context.Context) (done chan struct{}, err error)
}
