package roadaccidents

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/tracing"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type RoadAccidentSvc interface {
	Start(ctx context.Context) error
	getAndPublishRoadAccidents(ctx context.Context, lastChangeID string) (string, error)
	getRoadAccidentsFromTFV(ctx context.Context, lastChangeID string) ([]byte, error)
	publishRoadAccidentsToContextBroker(ctx context.Context, dev tfvDeviation) error
	updateRoadAccidentStatus(ctx context.Context, dev tfvDeviation) error
}

type ts struct {
	authKey          string
	tfvURL           string
	contextBrokerURL string
}

var tracer = otel.Tracer("roadaccidents")

func NewService(authKey, tfvURL, contextBrokerURL string) RoadAccidentSvc {
	return &ts{
		authKey:          authKey,
		tfvURL:           tfvURL,
		contextBrokerURL: contextBrokerURL,
	}
}

func (ts *ts) Start(ctx context.Context) error {
	var err error
	lastChangeID := "0"

	logger := logging.GetLoggerFromContext(ctx)

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

	_, ctx, _ = addTraceIDToLoggerAndStoreInContext(span, logging.GetLoggerFromContext(ctx), ctx)

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
				_, exists := previousDeviations[dev.Id]
				if exists {
					continue
				}

				err = ts.publishRoadAccidentsToContextBroker(ctx, dev)
				if err != nil {
					log.Error().Err(err).Msgf("failed to publish road accident %s: %s", dev.Id, err.Error())
					continue
				}

				previousDeviations[dev.Id] = dev.Id

			}
		} else {
			for _, dev := range sitch.Deviation {
				err = ts.updateRoadAccidentStatus(ctx, dev)
				if err != nil {
					log.Error().Err(err).Msgf("failed to update road accident %s: %s", dev.Id, err.Error())
					continue
				}
			}
		}
	}

	return tfvResp.Response.Result[0].Info.LastChangeID, err
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
