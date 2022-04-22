package trafficsvc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/rs/zerolog"
)

type TrafficService interface {
	Start(ctx context.Context) error
	getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error)
	getRoadAccidentsFromTFV(ctx context.Context, lastChangeID string) ([]byte, error)
	publishRoadAccidentsToContextBroker(ctx context.Context, dev tfvDeviation) error
	updateRoadAccidentStatus(ctx context.Context, dev tfvDeviation) error
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
	var err error
	lastChangeID := "0"

	for {
		lastChangeID, err = ts.getAndPublishRoadAccidents(ctx, lastChangeID)
		if err != nil {
			ts.log.Error().Msg(err.Error())
			return err
		}

		time.Sleep(30 * time.Second)
	}
}

func (ts *ts) getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error) {
	resp, err := ts.getRoadAccidentsFromTFV(ctx, lastChangeID)
	if err != nil {
		return lastChangeID, err
	}

	tfvResp := &tfvResponse{}
	err = json.Unmarshal(resp, tfvResp)
	if err != nil {
		return lastChangeID, err
	}

	for _, sitch := range tfvResp.Response.Result[0].Situation {
		if !sitch.Deleted {
			for _, dev := range sitch.Deviation {
				err = ts.publishRoadAccidentsToContextBroker(ctx, dev)
				if err != nil {
					return lastChangeID, err
				}
			}
		} else {
			for _, dev := range sitch.Deviation {
				err = ts.updateRoadAccidentStatus(ctx, dev)
				if err != nil {
					return lastChangeID, err
				}
			}
		}
	}

	return tfvResp.Response.Result[0].Info.LastChangeID, err
}

func (ts *ts) updateRoadAccidentStatus(ctx context.Context, dev tfvDeviation) error {
	return nil
}
