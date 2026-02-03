package weathersvc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var httpClient = http.Client{
	Transport: otelhttp.NewTransport(http.DefaultTransport),
	Timeout:   10 * time.Second,
}

func (ws *weatherSvc) getWeatherMeasurepointStatus(ctx context.Context, lastChangeID string) ([]byte, error) {
	var err error

	ctx, span := tracer.Start(ctx, "get-weathermeasurepoints")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	requestBody := fmt.Sprintf("<REQUEST><LOGIN authenticationkey=\"%s\" /><QUERY objecttype=\"WeatherMeasurepoint\" schemaversion=\"2.1\" changeid=\"%s\"><INCLUDE>Deleted</INCLUDE><INCLUDE>Id</INCLUDE><INCLUDE>Geometry.WGS84</INCLUDE><INCLUDE>Observation.Air.RelativeHumidity.Value</INCLUDE><INCLUDE>Observation.Air.Temperature.Value</INCLUDE><INCLUDE>Observation.Wind.Direction.Value</INCLUDE><INCLUDE>Observation.Sample</INCLUDE><INCLUDE>ModifiedTime</INCLUDE><INCLUDE>Name</INCLUDE><FILTER><WITHIN name=\"Geometry.SWEREF99TM\" shape=\"box\" value=\"%s\" /></FILTER></QUERY></REQUEST>", ws.authenticationKey, lastChangeID, ws.weatherBox)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ws.trafikverketURL, bytes.NewBufferString(requestBody))
	if err != nil {
		err = fmt.Errorf("failed to create http request: %s", err.Error())
		return []byte{}, err
	}
	apiReq.Header.Set("Content-Type", "text/xml")

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		err = fmt.Errorf("failed to retrieve weathermeasurepoints: %s", err.Error())
		return nil, err
	}
	defer apiResponse.Body.Close()

	if apiResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return []byte{}, err
	}

	return io.ReadAll(apiResponse.Body)
}
