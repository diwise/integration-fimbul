package main

import (
	"context"
	"flag"
	"strings"

	"github.com/diwise/context-broker/pkg/ngsild/client"
	"github.com/diwise/integration-fimbul/internal/application"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
)

var stationIds string

func main() {
	flag.StringVar(&stationIds, "stations", "", "id of the station(s) to retrieve data from")
	flag.Parse()

	serviceName := "integration-fimbul"
	serviceVersion := buildinfo.SourceVersion()
	ctx, log, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	service := env.GetVariableOrDie(log, "FIMBUL_URL", "url to weather service")
	woPrefix := env.GetVariableOrDie(log, "WO_PREFIX", "prefix ending for the entity id")
	ctxBrokerURL := env.GetVariableOrDie(log, "CONTEXT_BROKER_URL", "url to context broker")

	ctxBroker := client.NewContextBrokerClient(ctxBrokerURL, client.Debug("true"))

	app := application.New(ctxBroker, service)

	err := app.CreateWeatherObserved(ctx, woPrefix, func() []application.StationID {
		var stations []application.StationID

		stationList := strings.Split(stationIds, ",")

		for _, s := range stationList {
			if s != "" {
				stations = append(stations, application.StationID(s))
			}
		}

		return stations
	})

	if err != nil {
		log.Error().Err(err).Msg("program failed")
	}
}
