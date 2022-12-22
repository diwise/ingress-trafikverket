package roadaccidents

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var ErrAlreadyExists = errors.New("already exists")

type RoadAccidentSvc interface {
	Start(ctx context.Context) error
	getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error)
	getRoadAccidentsFromTFV(ctx context.Context, lastChangeID string) ([]byte, error)
	publishRoadAccidentToContextBroker(ctx context.Context, dev tfvDeviation) error
	updateRoadAccidentStatus(ctx context.Context, dev tfvDeviation) error
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
			logger.Error().Err(err).Msg(err.Error())
		}
	}
}

var previousDeviations map[string]string = make(map[string]string)

func (ts *ts) getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-and-publish")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	_, ctx, _ = addTraceIDToLoggerAndStoreInContext(span, logging.GetFromContext(ctx), ctx)

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
		if !sitch.Deleted { // check if this if can be moved into publishRoadAccidentToContextBroker with the new pattern of attempting merge before create.
			for _, dev := range sitch.Deviation {
				if dev.IconId == DeviationTypeRoadAccident {
					err = ts.publishRoadAccidentToContextBroker(ctx, dev)
					if err != nil && !errors.Is(err, ErrAlreadyExists) {
						log.Error().Err(err).Msgf("failed to publish road accident %s", dev.Id)
						continue
					}

					previousDeviations[dev.Id] = dev.Id
				} else {
					log.Info().Msgf("ignoring deviation of type %s", dev.IconId)
				}
			}
		} else {
			for _, dev := range sitch.Deviation {
				_, exists := previousDeviations[dev.Id]

				if exists {
					err = ts.updateRoadAccidentStatus(ctx, dev)
					if err != nil {
						log.Error().Err(err).Msgf("failed to update road accident %s", dev.Id)
						continue
					}
				}
			}
		}
	}

	return tfvResp.Response.Result[0].Info.LastChangeID, nil
}

func addTraceIDToLoggerAndStoreInContext(span trace.Span, logger zerolog.Logger, ctx context.Context) (string, context.Context, zerolog.Logger) {

	log := logger
	traceID := span.SpanContext().TraceID()
	traceIDStr := ""

	if traceID.IsValid() {
		traceIDStr = traceID.String()
		log = log.With().Str("traceID", traceIDStr).Logger()
	}

	ctx = logging.NewContextWithLogger(ctx, log)
	return traceIDStr, ctx, log
}
