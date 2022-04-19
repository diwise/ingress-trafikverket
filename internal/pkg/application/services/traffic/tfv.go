package trafficsvc

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

var tfvtracer = otel.Tracer("tfv-trafficinfo-client")

func (ts *ts) getTrafficInformationFromTFV(ctx context.Context) ([]byte, error) {
	var err error
	ctx, span := tfvtracer.Start(ctx, "get-tfv-traffic-information")
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

	requestBody := fmt.Sprintf(`<REQUEST>
	<LOGIN authenticationkey="%s" />
	<QUERY objecttype="Situation" schemaversion="1.2">
		  <FILTER>
		  		<EQ name="Deviation.CountyNo" value="2281" />
				<EQ name="Deviation.MessageType" value="Trafikmeddelande,Olycka" />
		  </FILTER>
		  <INCLUDE>Deviation.Id</INCLUDE>
		  <INCLUDE>Deviation.StartTime</INCLUDE>
		  <INCLUDE>Deviation.EndTime</INCLUDE>
		  <INCLUDE>Deviation.Message</INCLUDE>
		  <INCLUDE>Deviation.IconId</INCLUDE>
		  <INCLUDE>Deviation.Geometry.WGS84</INCLUDE>
	</QUERY>
</REQUEST>`, ts.authKey)

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
	if apiResponse.StatusCode != http.StatusOK {
		log.Error().Msgf("failed to retrieve traffic information, expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return nil, errors.New("")
	}

	defer apiResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(apiResponse.Body)

	log.Info().Msgf("received response: " + string(responseBody))

	return responseBody, err
}
