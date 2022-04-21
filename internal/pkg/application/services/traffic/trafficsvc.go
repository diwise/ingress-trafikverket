package trafficsvc

import (
	"context"
	"time"

	"github.com/rs/zerolog"
)

type TrafficService interface {
	Start(ctx context.Context) error
	getRoadAccidentsFromTFV(ctx context.Context) ([]byte, error)
	publishRoadAccidentsToContextBroker(resp []byte, ctx context.Context) error
}

type ts struct {
	log              zerolog.Logger
	authKey          string
	tfvURL           string
	contextBrokerURL string
}

func NewTrafficService(log zerolog.Logger, authKey, tfvURL, contextBrokerURL string) TrafficService {
	return &ts{
		log:              log,
		authKey:          authKey,
		tfvURL:           tfvURL,
		contextBrokerURL: contextBrokerURL,
	}
}

func (ts *ts) Start(ctx context.Context) error {
	for {
		resp, err := ts.getRoadAccidentsFromTFV(ctx)
		if err != nil {
			ts.log.Error().Msg(err.Error())
			return err
		}

		err = ts.publishRoadAccidentsToContextBroker(resp, ctx)
		if err != nil {
			ts.log.Error().Msg(err.Error())
			return err
		}

		time.Sleep(30 * time.Second)
	}
}
