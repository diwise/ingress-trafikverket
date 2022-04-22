package main

import (
	"context"
	"os"
	"runtime/debug"

	roadaccidents "github.com/diwise/ingress-trafikverket/internal/pkg/application/services/traffic"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/logging"
	"github.com/diwise/ingress-trafikverket/internal/pkg/infrastructure/tracing"
	"github.com/rs/zerolog"
)

func main() {
	serviceVersion := version()
	serviceName := "ingress-trafikverket"

	ctx, logger := logging.NewLogger(context.Background(), serviceName, serviceVersion)
	logger.Info().Msg("starting up ...")

	cleanup, err := tracing.Init(ctx, logger, serviceName, serviceVersion)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init tracing")
	}
	defer cleanup()

	authenticationKey := getEnvironmentVariableOrDie(logger, "TFV_API_AUTH_KEY", "API Authentication Key")
	trafikverketURL := getEnvironmentVariableOrDie(logger, "TFV_API_URL", "API URL")
	/*contextBrokerURL := getEnvironmentVariableOrDie(logger, "CONTEXT_BROKER_URL", "Context Broker URL")

	ws := weathersvc.NewWeatherService(logger, authenticationKey, trafikverketURL, contextBrokerURL)
	go ws.Start(ctx)*/

	ts := roadaccidents.NewTrafficService(logger, authenticationKey, trafikverketURL, "")
	ts.Start(ctx)
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
		log.Fatal().Msgf("Please set %s to a valid %s.", envVar, description)
	}
	return value
}
