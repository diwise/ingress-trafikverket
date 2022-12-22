package roadaccidents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/diwise/ngsi-ld-golang/pkg/datamodels/fiware"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (ts *ts) updateRoadAccidentStatus(ctx context.Context, dev tfvDeviation) error {
	var err error
	ctx, span := tracer.Start(ctx, "patch-entity-status")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ra := fiware.NewRoadAccident(dev.Id)
	ra.Status = *ngsitypes.NewTextProperty("solved")
	ra.DateModified = ngsitypes.CreateDateTimeProperty(time.Now().UTC().Format(time.RFC3339))

	patchBody, err := json.Marshal(ra)
	if err != nil {
		return err
	}

	cbUrl := fmt.Sprintf("%s/ngsi-ld/v1/entities/%s/attrs/", "", url.QueryEscape(ra.ID))

	req, _ := http.NewRequestWithContext(ctx, http.MethodPatch, cbUrl, bytes.NewBuffer(patchBody))
	req.Header.Add("Content-Type", "application/ld+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusMultiStatus {
		errMsg := fmt.Sprintf(
			"failed to update road accident %s in context broker, expected status code %d, but got %d",
			dev.Id, http.StatusNoContent, resp.StatusCode,
		)
		return errors.New(errMsg)
	}

	logger := logging.GetFromContext(ctx)
	traceID, _ := tracing.ExtractTraceID(span)

	logger.Info().Str("traceID", traceID).Str("entityID", ra.ID).Msg("updated status to solved")

	return nil
}
