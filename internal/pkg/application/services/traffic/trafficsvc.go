package trafficsvc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

type TrafficService interface {
	Start(ctx context.Context) error
	getTrafficInformation(ctx context.Context) ([]byte, error)
}

type ts struct {
	log     zerolog.Logger
	authKey string
	tfvURL  string
}

func NewTrafficService(log zerolog.Logger, authKey, tfvURL string) TrafficService {
	return &ts{
		log:     log,
		authKey: authKey,
		tfvURL:  tfvURL,
	}
}

func (ts *ts) Start(ctx context.Context) error {
	for {
		_, err := ts.getTrafficInformation(ctx)
		if err != nil {
			ts.log.Error().Msg(err.Error())
			return err
		}

		time.Sleep(30 * time.Second)
	}
}

var tracer = otel.Tracer("tfv-trafficinfo-client")

func (ts *ts) getTrafficInformation(ctx context.Context) ([]byte, error) {
	var err error
	ctx, span := tracer.Start(ctx, "get-trafficinformation")
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
					<EQ name="Deviation.MessageType" value="Olycka" />
					<EQ name="Deviation.CountyNo" value="2281" />
			</FILTER>
			<INCLUDE>Deviation.Id</INCLUDE>
			<INCLUDE>Deviation.Header</INCLUDE>
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
		return []byte{}, errors.New("")
	}

	defer apiResponse.Body.Close()

	responseBody, err := ioutil.ReadAll(apiResponse.Body)

	log.Info().Msgf("received response: " + string(responseBody))

	return responseBody, err
}
