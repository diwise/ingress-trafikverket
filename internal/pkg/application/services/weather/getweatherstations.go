package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("tfv-weatherstation-client")

func (ws *ws) getWeatherStationStatus(ctx context.Context, lastChangeID string) ([]byte, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-weatherstations")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	log := logging.GetLoggerFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	requestBody := fmt.Sprintf("<REQUEST><LOGIN authenticationkey=\"%s\" /><QUERY objecttype=\"WeatherStation\" schemaversion=\"1\" changeid=\"%s\"><INCLUDE>Id</INCLUDE><INCLUDE>Geometry.WGS84</INCLUDE><INCLUDE>Measurement.Air.Temp</INCLUDE><INCLUDE>Measurement.MeasureTime</INCLUDE><INCLUDE>ModifiedTime</INCLUDE><INCLUDE>Name</INCLUDE><FILTER><WITHIN name=\"Geometry.SWEREF99TM\" shape=\"box\" value=\"527000 6879000, 652500 6950000\" /></FILTER></QUERY></REQUEST>", ws.authenticationKey, lastChangeID)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ws.trafikverketURL, bytes.NewBufferString(requestBody))
	if err != nil {
		log.Error().Err(err).Msg("failed to create http request")
		return []byte{}, err
	}
	apiReq.Header.Set("Content-Type", "text/xml")

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		log.Error().Msgf("failed to retrieve weatherstations")
		return nil, err
	}
	if apiResponse.StatusCode != http.StatusOK {
		log.Error().Msgf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return []byte{}, errors.New("")
	}

	defer apiResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(apiResponse.Body)

	log.Info().Msgf("received response: " + string(responseBody))

	return responseBody, err
}
