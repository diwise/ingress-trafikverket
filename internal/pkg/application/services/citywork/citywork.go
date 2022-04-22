package citywork

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

var sdltracer = otel.Tracer("sdl-trafficinfo-client")

type CityWorkSvc interface {
	Start(ctx context.Context) error
	getTrafficInformationFromSDL(ctx context.Context) ([]byte, error)
}

func NewCityWorkService(log zerolog.Logger, sundsvallvaxerURL string, contextBrokerURL string) CityWorkSvc {
	return &cw{
		log:               log,
		sundsvallvaxerURL: sundsvallvaxerURL,
		contextBrokerURL:  contextBrokerURL,
	}
}

type cw struct {
	log               zerolog.Logger
	sundsvallvaxerURL string
	contextBrokerURL  string
}

func (cw *cw) Start(ctx context.Context) error {
	var err error

	for {
		r, err = cw.getTrafficInformationFromSDL(ctx)
		if err != nil {
			cw.log.Error().Msg(err.Error())
			return err
		}
		time.Sleep(30 * time.Second)
	}
}

func (cw *cw) getTrafficInformationFromSDL(ctx context.Context) ([]byte, error) {
	var err error
	ctx, span := sdltracer.Start(ctx, "get-sdl-traffic-information")
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

	apiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://karta.sundsvall.se/origoserver/converttogeojson/?q=sundsvallvaxerGC", nil)
	if err != nil {
		return nil, err
	}

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
