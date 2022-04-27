package roadaccidents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/fiware"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
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
		errMsg := fmt.Sprintf("failed to send road accident to context broker, expected status code %d, but got %d", http.StatusNoContent, resp.StatusCode)
		return errors.New(errMsg)
	}

	return nil
}
