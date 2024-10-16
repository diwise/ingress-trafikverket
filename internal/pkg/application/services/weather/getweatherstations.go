package weathersvc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (ws *ws) getWeatherStationStatus(ctx context.Context, lastChangeID string) ([]byte, error) {
	var err error

	ctx, span := tracer.Start(ctx, "get-weatherstations")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	requestBody := fmt.Sprintf("<REQUEST><LOGIN authenticationkey=\"%s\" /><QUERY objecttype=\"WeatherStation\" schemaversion=\"1\" changeid=\"%s\"><INCLUDE>Active</INCLUDE><INCLUDE>Id</INCLUDE><INCLUDE>Geometry.WGS84</INCLUDE><INCLUDE>Measurement.Air.RelativeHumidity</INCLUDE><INCLUDE>Measurement.Air.Temp</INCLUDE><INCLUDE>Measurement.Precipitation.Amount</INCLUDE><INCLUDE>Measurement.Wind.Direction</INCLUDE><INCLUDE>Measurement.Wind.Force</INCLUDE><INCLUDE>Measurement.Wind.ForceMax</INCLUDE><INCLUDE>Measurement.MeasureTime</INCLUDE><INCLUDE>ModifiedTime</INCLUDE><INCLUDE>Name</INCLUDE><FILTER><WITHIN name=\"Geometry.SWEREF99TM\" shape=\"box\" value=\"527000 6879000, 652500 6950000\" /></FILTER></QUERY></REQUEST>", ws.authenticationKey, lastChangeID)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ws.trafikverketURL, bytes.NewBufferString(requestBody))
	if err != nil {
		log.Error("failed to create http request", "err", err.Error())
		return []byte{}, err
	}
	apiReq.Header.Set("Content-Type", "text/xml")

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		log.Error("failed to retrieve weatherstations", "err", err.Error())
		return nil, err
	}
	if apiResponse.StatusCode != http.StatusOK {
		log.Error(fmt.Sprintf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode))
		return []byte{}, errors.New("")
	}

	defer apiResponse.Body.Close()

	responseBody, err := io.ReadAll(apiResponse.Body)

	log.Debug("received response", "body", string(responseBody))

	return responseBody, err
}
