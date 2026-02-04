package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/ingress-trafikverket/internal/pkg/application/services"
	"github.com/diwise/ingress-trafikverket/internal/pkg/application/services/roadaccidents"
	weathersvc "github.com/diwise/ingress-trafikverket/internal/pkg/application/services/weather"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

const serviceName string = "ingress-trafikverket"

func main() {
	serviceVersion := buildinfo.SourceVersion()
	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion, "json")
	defer cleanup()

	authenticationKey := env.GetVariableOrDie(ctx, "TFV_API_AUTH_KEY", "API authentication key")
	trafikverketURL := env.GetVariableOrDie(ctx, "TFV_API_URL", "API URL")
	countyCode := env.GetVariableOrDefault(ctx, "TFV_COUNTY_CODE", "")
	weatherBox := env.GetVariableOrDefault(ctx, "TFV_WEATHER_BOX", "527000 6879000, 652500 6950000")
	contextBrokerURL := env.GetVariableOrDie(ctx, "CONTEXT_BROKER_URL", "context broker URL")
	ctxBrokerClient := client.NewContextBrokerClient(contextBrokerURL, client.Debug("true"))

	ctx, stopAllServices := context.WithCancel(ctx)

	services := createServices(ctx, authenticationKey, trafikverketURL, countyCode, weatherBox, ctxBrokerClient)

	var wg sync.WaitGroup

	for _, svc := range services {
		wg.Add(1)
		go func() {
			done, err := svc.Start(ctx)
			if err != nil {
				logger.Error("failed to start service", "err", err.Error())
				os.Exit(1)
			}
			<-done
			wg.Done()
		}()
	}

	apiPort := env.GetVariableOrDefault(ctx, "SERVICE_PORT", "8080")
	mux := setupServeMux(ctx)
	webServer := &http.Server{Addr: ":" + apiPort, Handler: mux}

	go func() {
		if err := webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start request router", "err", err.Error())
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigChan

	logger.Debug("received signal", "signal", s)

	stopAllServices()

	logger.Info("waiting for all services to shut down...")
	wg.Wait()

	err := webServer.Shutdown(ctx)
	if err != nil {
		logger.Error("failed to shutdown web server", "err", err.Error())
	}

	logger.Info("shutting down")
}

func createServices(ctx context.Context, authenticationKey, trafikverketURL, countyCode, weatherBox string, ctxBrokerClient client.ContextBrokerClient) []services.Starter {
	services := make([]services.Starter, 0, 2)
	logger := logging.GetFromContext(ctx)

	if featureIsEnabled(logger, "weather") {
		services = append(
			services,
			weathersvc.NewWeatherService(ctx, authenticationKey, trafikverketURL, weatherBox, ctxBrokerClient),
		)
	}

	if featureIsEnabled(logger, "roadaccident") {
		services = append(
			services,
			roadaccidents.NewService(ctx, authenticationKey, trafikverketURL, countyCode, ctxBrokerClient),
		)
	}

	return services
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

func setupServeMux(_ context.Context) *http.ServeMux {
	r := http.NewServeMux()

	r.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	return r
}
