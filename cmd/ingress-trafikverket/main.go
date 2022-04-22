package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/diwise/ingress-trafikverket/internal/pkg/application/services/roadaccidents"
	weathersvc "github.com/diwise/ingress-trafikverket/internal/pkg/application/services/weather"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/tracing"
	"github.com/rs/zerolog"
)

const serviceName string = "ingress-trafikverket"

func main() {
	serviceVersion := version()

	ctx, logger := logging.NewLogger(context.Background(), serviceName, serviceVersion)
	logger.Info().Msg("starting up ...")

	cleanup, err := tracing.Init(ctx, logger, serviceName, serviceVersion)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init tracing")
	}
	defer cleanup()

	authenticationKey := getEnvironmentVariableOrDie(logger, "TFV_API_AUTH_KEY", "API Authentication Key")
	trafikverketURL := getEnvironmentVariableOrDie(logger, "TFV_API_URL", "API URL")
	contextBrokerURL := getEnvironmentVariableOrDie(logger, "CONTEXT_BROKER_URL", "Context Broker URL")

	if featureIsEnabled(logger, "weather") {
		ws := weathersvc.NewWeatherService(logger, authenticationKey, trafikverketURL, contextBrokerURL)
		go ws.Start(ctx)
	}

	if featureIsEnabled(logger, "roadaccident") {
		ts := roadaccidents.NewRoadAccidentSvc(logger, authenticationKey, trafikverketURL, contextBrokerURL)
		go ts.Start(ctx)
	}

	for {
		time.Sleep(5 * time.Second)
	}
}

//featureIsEnabled checks wether a given feature is enabled by exanding the feature name into <uppercase>_ENABLED and checking if the corresponding environment variable is set to true.
//  Ex: weather -> WEATHER_ENABLED
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

func version() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	buildSettings := buildInfo.Settings
	infoMap := map[string]string{}
	for _, s := range buildSettings {
		infoMap[s.Key] = s.Value
	}

	sha := infoMap["vcs.revision"]
	if infoMap["vcs.modified"] == "true" {
		sha += "+"
	}

	return sha
}

func getEnvironmentVariableOrDie(log zerolog.Logger, envVar, description string) string {
	value := os.Getenv(envVar)
	if value == "" {
		log.Fatal().Msgf("please set %s to a valid %s.", envVar, description)
	}
	return value
}
