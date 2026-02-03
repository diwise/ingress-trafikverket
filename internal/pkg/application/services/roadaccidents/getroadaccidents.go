package roadaccidents

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

func (ts *roadAccidentSvc) getRoadAccidentsFromTFV(ctx context.Context, lastChangeID string) ([]byte, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-traffic-information")
	defer func() { tracing.RecordAnyErrorAndEndSpan(err, span) }()

	countyFilter := ""
	if len(ts.countyCode) > 0 {
		countyFilter = fmt.Sprintf("<EQ name=\"Deviation.CountyNo\" value=\"%s\" />", ts.countyCode)
	}

	requestBody := fmt.Sprintf(`<REQUEST>
	<LOGIN authenticationkey="%s" />
	<QUERY objecttype="Situation" namespace="road.trafficinfo" schemaversion="1.6" changeid="%s" includedeletedobjects="true">
		  <FILTER>
			  <EQ name="Deviation.MessageType" value="Olycka" />%s
		  </FILTER>
		  <INCLUDE>Deviation.Id</INCLUDE>
		  <INCLUDE>Deviation.StartTime</INCLUDE>
		  <INCLUDE>Deviation.EndTime</INCLUDE>
		  <INCLUDE>Deviation.Message</INCLUDE>
		  <INCLUDE>Deviation.IconId</INCLUDE>
		  <INCLUDE>Deviation.Geometry.Point.WGS84</INCLUDE>
		  <INCLUDE>Deleted</INCLUDE>
	</QUERY>
</REQUEST>`, ts.authKey, lastChangeID, countyFilter)

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.tfvURL, bytes.NewBufferString(requestBody))
	if err != nil {
		err = fmt.Errorf("failed to create request: %s", err.Error())
		return nil, err
	}
	apiReq.Header.Set("Content-Type", "text/xml")

	apiResponse, err := httpClient.Do(apiReq)
	if err != nil {
		err = fmt.Errorf("request for traffic information failed: %s", err.Error())
		return nil, err
	}
	defer apiResponse.Body.Close()

	if apiResponse.StatusCode != http.StatusOK {
		err = fmt.Errorf("failed to retrieve traffic information, expected status code %d, but got %d", http.StatusOK, apiResponse.StatusCode)
		return nil, err
	}

	return io.ReadAll(apiResponse.Body)
}
