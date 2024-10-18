package roadaccidents

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/ingress-trafikverket/internal/pkg/application/services"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

var ErrAlreadyExists = errors.New("already exists")

type RoadAccidentSvc interface {
	services.Starter
}

type roadAccidentSvc struct {
	authKey    string
	tfvURL     string
	countyCode string

	interval time.Duration

	ctxBroker client.ContextBrokerClient
}

var tracer = otel.Tracer("roadaccidents")

func NewService(_ context.Context, authKey, tfvURL, countyCode string, ctxBroker client.ContextBrokerClient) RoadAccidentSvc {
	return &roadAccidentSvc{
		authKey:    authKey,
		tfvURL:     tfvURL,
		countyCode: countyCode,
		interval:   30 * time.Second,
		ctxBroker:  ctxBroker,
	}
}

func (ras *roadAccidentSvc) Start(ctx context.Context) (chan struct{}, error) {

	done := make(chan struct{})

	go func() {
		var err error
		lastChangeID := "0"

		tmr := time.NewTicker(ras.interval)

		defer func() {
			tmr.Stop()
			done <- struct{}{}
		}()

		for {
			select {
			case <-tmr.C:
				{
					lastChangeID, err = ras.getAndPublishRoadAccidents(ctx, lastChangeID)
					if err != nil {
						logger := logging.GetFromContext(ctx)
						logger.Error("failed to get and publish road accidents", "err", err.Error())
					}
				}
			case <-ctx.Done():
				{
					return
				}
			}
		}
	}()

	return done, nil
}

func (ras *roadAccidentSvc) getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-and-publish")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, logger := o11y.AddTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

	resp, err := ras.getRoadAccidentsFromTFV(ctx, lastChangeID)
	if err != nil {
		return lastChangeID, err
	}

	logger.Debug("received response", "body", string(resp))

	tfvResp := &tfvResponse{}
	err = json.Unmarshal(resp, tfvResp)
	if err != nil {
		return lastChangeID, err
	}

	for _, sitch := range tfvResp.Response.Result[0].Situation {
		for _, dev := range sitch.Deviation {
			if dev.IconId == DeviationTypeRoadAccident {
				err = ras.publishRoadAccidentToContextBroker(ctx, dev, sitch.Deleted)
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
