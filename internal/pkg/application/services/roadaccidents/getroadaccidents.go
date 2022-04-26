package roadaccidents

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (ts *ts) getRoadAccidentsFromTFV(ctx context.Context, lastChangeID string) ([]byte, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-traffic-information")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	log := logging.GetLoggerFromContext(ctx)

	httpClient := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	requestBody := fmt.Sprintf(`<REQUEST>
	<LOGIN authenticationkey="%s" />
	<QUERY objecttype="Situation" schemaversion="1.2" changeid="%s" includedeletedobjects="true">
		  <FILTER>
				<EQ name="Deviation.MessageType" value="Olycka" />
		  </FILTER>
		  <INCLUDE>Deviation.Id</INCLUDE>
		  <INCLUDE>Deviation.StartTime</INCLUDE>
		  <INCLUDE>Deviation.EndTime</INCLUDE>
		  <INCLUDE>Deviation.Message</INCLUDE>
		  <INCLUDE>Deviation.IconId</INCLUDE>
		  <INCLUDE>Deviation.Geometry.WGS84</INCLUDE>
		  <INCLUDE>Deleted</INCLUDE>
	</QUERY>
</REQUEST>`, ts.authKey, lastChangeID)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.tfvURL, bytes.NewBufferString(requestBody))
	if err != nil {
		return nil, err
	}
	apiReq.Header.Set("Content-Type", "text/xml")

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		log.Error().Msgf("failed to retrieve traffic information")
		return nil, err
	}
	defer apiResponse.Body.Close()

	if apiResponse.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("failed to retrieve traffic information, expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return nil, errors.New(errMsg)
	}

	responseBody, err := ioutil.ReadAll(apiResponse.Body)

	log.Info().Msgf("received response: " + string(responseBody))

	return responseBody, err
}
