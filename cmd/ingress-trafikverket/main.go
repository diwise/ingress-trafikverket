package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/ingress-trafikverket/internal/pkg/application/services/roadaccidents"
	weathersvc "github.com/diwise/ingress-trafikverket/internal/pkg/application/services/weather"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/rs/zerolog"
)

const serviceName string = "ingress-trafikverket"

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	authenticationKey := env.GetVariableOrDie(logger, "TFV_API_AUTH_KEY", "API authentication key")
	trafikverketURL := env.GetVariableOrDie(logger, "TFV_API_URL", "API URL")
	countyCode := env.GetVariableOrDefault(logger, "TFV_COUNTY_CODE", "")
	contextBrokerURL := env.GetVariableOrDie(logger, "CONTEXT_BROKER_URL", "context broker URL")
	ctxBrokerClient := client.NewContextBrokerClient(contextBrokerURL, client.Debug("true"))

	if featureIsEnabled(logger, "weather") {
		ws := weathersvc.NewWeatherService(logger, authenticationKey, trafikverketURL, contextBrokerURL, ctxBrokerClient)
		go ws.Start(ctx)
	}

	if featureIsEnabled(logger, "roadaccident") {
		ts := roadaccidents.NewService(authenticationKey, trafikverketURL, countyCode, contextBrokerURL)
		go ts.Start(ctx)
	}

	for {
		time.Sleep(5 * time.Second)
	}
}

// featureIsEnabled checks wether a given feature is enabled by exanding the feature name into <uppercase>_ENABLED and checking if the corresponding environment variable is set to true.
//
//	Ex: weather -> WEATHER_ENABLED
func featureIsEnabled(logger zerolog.Logger, feature string) bool {
	featureKey := fmt.Sprintf("%s_ENABLED", strings.ToUpper(feature))
	isEnabled := os.Getenv(featureKey) == "true"

	if isEnabled {
		logger.Info().Msgf("feature %s is enabled", feature)
	} else {
		logger.Warn().Msgf("feature %s is not enabled", feature)
	}

	return isEnabled
}
