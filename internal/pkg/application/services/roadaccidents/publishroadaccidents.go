package roadaccidents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/diwise/ingress-trafikverket/internal/pkg/fiware"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/tracing"
	"github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/geojson"
	ngsitypes "github.com/diwise/ngsi-ld-golang/pkg/ngsi-ld/types"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (ts *ts) publishRoadAccidentsToContextBroker(ctx context.Context, dev tfvDeviation) error {
	var err error
	ctx, span := tracer.Start(ctx, "publish-to-broker")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	logger := logging.GetLoggerFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	ra := fiware.NewRoadAccident(dev.Id)
	if dev.StartTime != "" {
		ra.AccidentDate = *ngsitypes.CreateDateTimeProperty(dev.StartTime)
	}
	if dev.Geometry.WGS84 != "" {
		ra.Location = getLocationFromString(dev.Geometry.WGS84)
	}

	ra.Description = *ngsitypes.NewTextProperty(dev.Message)
	ra.Status = *ngsitypes.NewTextProperty("onGoing")

	requestBody, err := json.Marshal(ra)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/ngsi-ld/v1/entities", ts.contextBrokerURL)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(requestBody))
	req.Header.Add("Content-Type", "application/ld+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusCreated {
		logger.Error().Msgf("failed to send road accident to context broker, expected status code %d, but got %d", http.StatusOK, resp.StatusCode)
		return errors.New("")
	}

	logger.Info().Msgf("publishing road accident %s to context broker: %s", ra.ID, string(requestBody))

	return err
}

func getLocationFromString(location string) *geojson.GeoJSONProperty {
	position := location[7 : len(location)-1]

	Longitude := strings.Split(position, " ")[0]
	newLong, _ := strconv.ParseFloat(Longitude, 32)
	Latitude := strings.Split(position, " ")[1]
	newLat, _ := strconv.ParseFloat(Latitude, 32)

	return geojson.CreateGeoJSONPropertyFromWGS84(newLong, newLat)
}
