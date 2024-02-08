package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/diwise/context-broker/pkg/datamodels/fiware"
	"github.com/diwise/context-broker/pkg/ngsild/client"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities"
	"github.com/diwise/context-broker/pkg/ngsild/types/entities/decorators"
	"github.com/diwise/context-broker/pkg/ngsild/types/properties"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Application interface {
	CreateWeatherObserved(ctx context.Context, prefixFormat string, stationIds func() []StationID) error
}

type StationID string

type app struct {
	cb      client.ContextBrokerClient
	service string
}

func New(cb client.ContextBrokerClient, service string) Application {
	return &app{
		cb:      cb,
		service: service,
	}
}

func (i app) CreateWeatherObserved(ctx context.Context, prefixEnding string, stationIds func() []StationID) error {
	log := logging.GetFromContext(ctx)
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	if !strings.HasSuffix(prefixEnding, ":") {
		prefixEnding = prefixEnding + ":"
	}

	stations := stationIds()
	if len(stations) == 0 {
		return errors.New("list of stations is empty")
	}

	for _, id := range stations {
		url := fmt.Sprintf("%s/stations/%s.last", i.service, id)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Error("failed to create request", "err", err.Error())
			return err
		}

		resp, err := client.Do(req)
		log.Info("requesting data", "station", id)
		if err != nil {
			log.Error("failed to send request", "err", err.Error())
			return err
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("request failed expected status code 200 but got: %d", resp.StatusCode)
		}

		ws := struct {
			Station weatherStation `json:"station"`
		}{}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Error("failed to read response body", "err", err.Error())
			return err
		}

		err = json.Unmarshal(bodyBytes, &ws)
		if err != nil {
			log.Error("failed to unmarshal response body into json", "err", err.Error())
			return err
		}

		entityID := fmt.Sprintf("%s%s%s", fiware.WeatherObservedIDPrefix, prefixEnding, ws.Station.ID)

		attributes, err := createWeatherObservedAttributes(ctx, ws.Station)
		if err != nil {
			log.Error("failed to create attributes for entity", "err", err.Error())
			return err
		}

		fragment, _ := entities.NewFragment(attributes...)

		headers := map[string][]string{"Content-Type": {"application/ld+json"}}

		log.Info("merging entity", "entityID", entityID)
		_, err = i.cb.MergeEntity(ctx, entityID, fragment, headers)
		if err != nil {
			if !errors.Is(err, ngsierrors.ErrNotFound) {
				log.Error("failed to merge entity", "entityID", entityID, "err", err.Error())
				return err
			}

			log.Info("entity not found, attempting create", "entityID", entityID)

			latitude, err := strconv.ParseFloat(ws.Station.Latitude, 64)
			if err != nil {
				log.Error("failed to parse latitude from string", "latitude", ws.Station.Latitude, "err", err.Error())
				return err
			}
			longitude, err := strconv.ParseFloat(ws.Station.Longitude, 64)
			if err != nil {
				log.Error("failed to parse longitude from string", "longitude", ws.Station.Longitude, "err", err.Error())
				return err
			}

			attributes = append(attributes, decorators.Location(latitude, longitude), decorators.Name(ws.Station.Name))

			entity, err := entities.New(entityID, fiware.WeatherObservedTypeName, attributes...)
			if err != nil {
				log.Error("failed to construct new entity", "err", err.Error())
				return err
			}

			_, err = i.cb.CreateEntity(ctx, entity, headers)
			if err != nil {
				log.Error("failed to create entity", "err", err.Error())
				return err
			}
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

func createWeatherObservedAttributes(ctx context.Context, ws weatherStation) ([]entities.EntityDecoratorFunc, error) {
	if len(ws.Logg) > 0 {
		temp, err := strconv.ParseFloat(ws.Logg[0].Temperature, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse temperature: %s", err.Error())
		}

		windSpeed, err := strconv.ParseFloat(ws.Logg[0].WindAverageSpeed, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse wind speed: %s", err.Error())
		}

		windDirection, err := strconv.ParseFloat(ws.Logg[0].WindDirection, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse wind direction: %s", err.Error())
		}

		relHum, err := strconv.ParseFloat(ws.Logg[0].RelativeHumidity, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse relative humidity: %s", err.Error())
		}

		layout := "2006-01-02 15:04:05"
		t, err := time.Parse(layout, ws.Logg[0].DateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse time from string: %s", err.Error())
		}

		for t.UTC().After(time.Now().UTC()) {
			t = t.Add(-1 * time.Hour)
		}

		utcTime := t.UTC().Format(time.RFC3339)

		attributes := append(
			make([]entities.EntityDecoratorFunc, 0, 7),
			decorators.Number("temperature", temp, properties.ObservedAt(utcTime)),
			decorators.Number("windSpeed", windSpeed, properties.ObservedAt(utcTime)),
			decorators.Number("windDirection", windDirection, properties.ObservedAt(utcTime)),
			decorators.Number("relativeHumidity", math.Round(relHum)/100, properties.ObservedAt(utcTime)),
			decorators.DateTime("dateObserved", utcTime),
		)

		return attributes, nil
	} else {
		wsBytes, _ := json.Marshal(ws)
		return nil, fmt.Errorf("weather station response does not contain logs: %s", wsBytes)
	}
}
