package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/ingress-trafikverket/internal/pkg/application/services/roadaccidents"
	weathersvc "github.com/diwise/ingress-trafikverket/internal/pkg/application/services/weather"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

const serviceName string = "ingress-trafikverket"

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	authenticationKey := env.GetVariableOrDie(ctx, "TFV_API_AUTH_KEY", "API authentication key")
	trafikverketURL := env.GetVariableOrDie(ctx, "TFV_API_URL", "API URL")
	countyCode := env.GetVariableOrDefault(ctx, "TFV_COUNTY_CODE", "")
	contextBrokerURL := env.GetVariableOrDie(ctx, "CONTEXT_BROKER_URL", "context broker URL")
	ctxBrokerClient := client.NewContextBrokerClient(contextBrokerURL, client.Debug("true"))

	if featureIsEnabled(logger, "weather") {
		ws := weathersvc.NewWeatherService(ctx, authenticationKey, trafikverketURL, ctxBrokerClient)
		go ws.Start(ctx)
	}

	if featureIsEnabled(logger, "roadaccident") {
		ts := roadaccidents.NewService(authenticationKey, trafikverketURL, countyCode, ctxBrokerClient)
		go ts.Start(ctx)
	}

	setupRouterAndWaitForConnections(ctx)
}

// featureIsEnabled checks wether a given feature is enabled by exanding the feature name into <uppercase>_ENABLED and checking if the corresponding environment variable is set to true.
//
//	Ex: weather -> WEATHER_ENABLED
func featureIsEnabled(logger *slog.Logger, feature string) bool {
	featureKey := fmt.Sprintf("%s_ENABLED", strings.ToUpper(feature))
	isEnabled := os.Getenv(featureKey) == "true"

	if isEnabled {
		logger.Info("feature is enabled", "feature", feature)
	} else {
		logger.Warn("feature is not enabled", "feature", feature)
	}

	return isEnabled
}

func setupRouterAndWaitForConnections(ctx context.Context) {
	r := chi.NewRouter()
	r.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		Debug:            false,
	}).Handler)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	err := http.ListenAndServe(":8080", r)
	if err != nil {
		logging.GetFromContext(ctx).Error("failed to start router", "err", err.Error())
		os.Exit(1)
	}
}
