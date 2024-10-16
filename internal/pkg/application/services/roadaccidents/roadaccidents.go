package roadaccidents

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

var ErrAlreadyExists = errors.New("already exists")

type RoadAccidentSvc interface {
	Start(ctx context.Context) error
	getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error)
	getRoadAccidentsFromTFV(ctx context.Context, lastChangeID string) ([]byte, error)
	publishRoadAccidentToContextBroker(ctx context.Context, dev tfvDeviation, deleted bool) error
}

type ts struct {
	authKey    string
	tfvURL     string
	countyCode string
	ctxBroker  client.ContextBrokerClient
}

var tracer = otel.Tracer("roadaccidents")

func NewService(authKey, tfvURL, countyCode string, ctxBroker client.ContextBrokerClient) RoadAccidentSvc {
	return &ts{
		authKey:    authKey,
		tfvURL:     tfvURL,
		countyCode: countyCode,
		ctxBroker:  ctxBroker,
	}
}

func (ts *ts) Start(ctx context.Context) error {
	var err error
	lastChangeID := "0"

	logger := logging.GetFromContext(ctx)

	for {
		time.Sleep(30 * time.Second)

		lastChangeID, err = ts.getAndPublishRoadAccidents(ctx, lastChangeID)
		if err != nil {
			logger.Error("failed to get and publish road accidents", "err", err.Error())
		}
	}
}

func (ts *ts) getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-and-publish")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

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
		for _, dev := range sitch.Deviation {
			if dev.IconId == DeviationTypeRoadAccident {
				err = ts.publishRoadAccidentToContextBroker(ctx, dev, sitch.Deleted)
				if err != nil && !errors.Is(err, ErrAlreadyExists) {
					logger.Error("failed to publish road accident", "id", dev.Id, "err", err.Error())
					continue
				}
			} else {
				logger.Info("ignoring deviation", "deviationtype", dev.IconId)
			}
		}

	}

	return tfvResp.Response.Result[0].Info.LastChangeID, nil
}
