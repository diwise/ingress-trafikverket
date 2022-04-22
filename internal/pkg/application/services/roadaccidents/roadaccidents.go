package roadaccidents

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/fiware"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/tracing"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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
		lastChangeID, err = ts.getAndPublishRoadAccidents(ctx, lastChangeID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get and publish accidents")
		}

		time.Sleep(30 * time.Second)
	}
}

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
	var err error
	ctx, span := tracer.Start(ctx, "patch-entity-status")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ra := fiware.NewRoadAccident(dev.Id)
	ra.Status = *ngsitypes.NewTextProperty("solved")

	patchBody, err := json.Marshal(ra)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/ngsi-ld/v1/entity/%s/attrs", ts.contextBrokerURL, ra.ID)

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(patchBody))
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
