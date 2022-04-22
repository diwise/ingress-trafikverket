package citywork

import (
	"context"
	"encoding/json"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/domain"
	"github.com/diwise/ingress-trafikverket/internal/pkg/fiware"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

var sdltracer = otel.Tracer("sdl-trafficinfo-client")

type CityWorkSvc interface {
	Start(ctx context.Context) error
	publishCityWorkToContextBroker(ctx context.Context, citywork fiware.CityWork) error
}

func NewCityWorkService(log zerolog.Logger, sundsvallvaxerURL string, contextBrokerURL string) CityWorkSvc {
	s := NewSdlClient(log, sundsvallvaxerURL)
	c := domain.NewContextBrokerClient(contextBrokerURL, log)

	return &cw{
		log:           log,
		sdlClient:     s,
		contextbroker: c,
	}
}

type cw struct {
	log           zerolog.Logger
	sdlClient     SdlClient
	contextbroker domain.ContextBrokerClient
}

func (cw *cw) Start(ctx context.Context) error {
	for {
		response, err := cw.sdlClient.Get(ctx)
		if err != nil {
			cw.log.Error().Msg(err.Error())
			return err
		}

		m, err := toModel(response)
		if err != nil {
			cw.log.Error().Msg(err.Error())
			return err
		}

		for _, f := range m.Features {
			cwModel := toCityWorkModel(f)
			err = cw.publishCityWorkToContextBroker(ctx, cwModel)
			if err != nil {
				cw.log.Error().Msg(err.Error())
				return err
			}
		}

		time.Sleep(30 * time.Second)
	}
}

func toModel(resp []byte) (*sdlResponse, error) {
	var m sdlResponse

	err := json.Unmarshal(resp, &m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}

func toCityWorkModel(sf sdlFeature) fiware.CityWork {

	entityID := sf.Properties.Description

	cw := fiware.NewCityWork(entityID)

	return cw
}

func (cw *cw) publishCityWorkToContextBroker(ctx context.Context, citywork fiware.CityWork) error {
	if err := cw.contextbroker.Post(ctx, citywork); err != nil {
		cw.log.Error().Err(err)
		return err
	}
	return nil
}
