package roadaccidents

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/fiware"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type RoadAccidentSvc interface {
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

func NewRoadAccidentSvc(log zerolog.Logger, authKey, tfvURL, contextBrokerURL string) RoadAccidentSvc {
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

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ra := fiware.NewRoadAccident(dev.Id)
	ra.Status = *ngsitypes.NewTextProperty("solved")

	url := fmt.Sprintf("%s/ngsi-ld/v1/entity/%s/attrs", ts.contextBrokerURL, ra.ID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, nil)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return err
	}

	return nil
}
