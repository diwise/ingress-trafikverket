package trafficsvc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/fiware"
	"github.com/rs/zerolog"
)

type TrafficService interface {
	Start(ctx context.Context) error
	getTrafficInformationFromTFV(ctx context.Context) ([]byte, error)
	sendToContextBroker(resp []byte) error
}

type ts struct {
	log     zerolog.Logger
	authKey string
	tfvURL  string
}

func NewTrafficService(log zerolog.Logger, authKey, tfvURL string) TrafficService {
	return &ts{
		log:     log,
		authKey: authKey,
		tfvURL:  tfvURL,
	}
}

func (ts *ts) Start(ctx context.Context) error {
	for {
		resp, err := ts.getTrafficInformationFromTFV(ctx)
		if err != nil {
			ts.log.Error().Msg(err.Error())
			return err
		}

		err = ts.sendToContextBroker(resp)
		if err != nil {
			ts.log.Error().Msg(err.Error())
			return err
		}

		time.Sleep(30 * time.Second)
	}
}

func (ts *ts) sendToContextBroker(resp []byte) error {
	if resp == nil {
		ts.log.Info().Msg("no new incidents to send to context broker")
		return nil
	}

	tfvResp := tfvResponse{}

	err := json.Unmarshal(resp, &tfvResp)
	if err != nil {
		return err
	}

	//response should be unmartialed to tfvResponse, then mapped into new RoadAccident, then forwarded to context broker.
	fiware.NewRoadAccident(tfvResp.Response.Result[0].Situation[0].Deviation[0].Id)
	//find out how to determine that a situation has been resolved, and subsequently patch that information to the context broker

	return err
}
