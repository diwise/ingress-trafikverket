package weathersvc

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/ingress-trafikverket/internal/pkg/application/services"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/otel"
)

type WeatherService interface {
	services.Starter
}

func NewWeatherService(ctx context.Context, authKey, trafikverketURL, weatherBox string, ctxBrokerClient client.ContextBrokerClient) WeatherService {
	return &weatherSvc{
		authenticationKey: authKey,
		trafikverketURL:   trafikverketURL,
		weatherBox:        weatherBox,
		ctxBrokerClient:   ctxBrokerClient,
		interval:          30 * time.Second,
		stations:          map[string]time.Time{},
	}
}

type weatherSvc struct {
	authenticationKey string
	trafikverketURL   string
	weatherBox        string
	ctxBrokerClient   client.ContextBrokerClient
	interval          time.Duration
	stations          map[string]time.Time
}

func (ws *weatherSvc) Start(ctx context.Context) (chan struct{}, error) {

	done := make(chan struct{})

	go func() {
		var err error
		lastChangeID := "0"

		tmr := time.NewTicker(ws.interval)

		defer func() {
			tmr.Stop()
			done <- struct{}{}
		}()

		for {
			select {
			case <-tmr.C:
				{
					lastChangeID, err = ws.getAndPublishWeatherMeasurepoints(ctx, lastChangeID)
					if err != nil {
						logging.GetFromContext(ctx).Error(
							"failed to get and publish weather stations", "err", err.Error(),
						)
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

var tracer = otel.Tracer("tfv-weathermeasurepoint-client")

func (ws *weatherSvc) getAndPublishWeatherMeasurepoints(ctx context.Context, lastChangeID string) (string, error) {
	var err error

	ctx, span := tracer.Start(ctx, "get-and-publish-status")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, log := o11y.AddTraceIDToLoggerAndStoreInContext(
		span, logging.GetFromContext(ctx), ctx,
	)

	responseBody, err := ws.getWeatherMeasurepointStatus(ctx, lastChangeID)
	if err != nil {
		return lastChangeID, err
	}

	log.Debug("received response", "body", string(responseBody))

	answer := &weatherMeasurepointResponse{}
	err = json.Unmarshal(responseBody, answer)
	if err != nil {
		return lastChangeID, err
	}

	for _, measurepoint := range answer.Response.Result[0].WeatherMeasurepoints {
		if !measurepoint.Deleted {
			previousMeasureTime, ok := ws.stations[measurepoint.ID]
			if ok && !measurepoint.ModifiedTime.After(previousMeasureTime) {
				continue
			}

			ws.stations[measurepoint.ID] = measurepoint.ModifiedTime

			err = ws.publishWeatherMeasurepointStatus(ctx, measurepoint)
			if err != nil {
				log.Error("unable to publish data for weathermeasurepoint", "measurepoint", measurepoint.ID, "err", err)
			}
		}
	}

	return answer.Response.Result[0].Info.LastChangeID, nil
}
