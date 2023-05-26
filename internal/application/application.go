package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

func (i app) CreateWeatherObserved(ctx context.Context, prefixFormat string, stationIds func() []StationID) error {
	log := logging.GetFromContext(ctx)
	client := http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}

	stations := stationIds()
	if len(stations) == 0 {
		return errors.New("list of stations is empty")
	}

	for _, id := range stations {
		url := fmt.Sprintf("%s/stations/%s.last", i.service, id)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Error().Err(err).Msg("failed to create request")
			return err
		}

		resp, err := client.Do(req)
		log.Info().Msgf("requesting data from station: %s", id)
		if err != nil {
			log.Error().Err(err).Msg("failed to send request")
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
			log.Error().Err(err).Msg("failed to read response body")
			return err
		}

		err = json.Unmarshal(bodyBytes, &ws)
		if err != nil {
			log.Error().Err(err).Msg("failed to unmarshal response body into json")
			return err
		}

		entityID := fmt.Sprintf("%s%s%s", fiware.WeatherObservedIDPrefix, prefixFormat, ws.Station.ID)

		attributes, err := createWeatherObservedAttributes(ctx, ws.Station)
		if err != nil {
			log.Error().Err(err).Msg("failed to create attributes for entity")
			return err
		}

		fragment, _ := entities.NewFragment(attributes...)

		headers := map[string][]string{"Content-Type": {"application/ld+json"}}

		log.Info().Msgf("merging entity %s", entityID)
		_, err = i.cb.MergeEntity(ctx, entityID, fragment, headers)
		if err != nil {
			if !errors.Is(err, ngsierrors.ErrNotFound) {
				log.Error().Err(err).Msg("failed to merge entity")
				return err
			}

			log.Info().Msgf("entity with id %s not found, attempting create", entityID)

			latitude, err := strconv.ParseFloat(ws.Station.Latitude, 64)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse latitude from string")
				return err
			}
			longitude, err := strconv.ParseFloat(ws.Station.Longitude, 64)
			if err != nil {
				log.Error().Err(err).Msg("failed to parse longitude from string")
				return err
			}

			attributes = append(attributes, decorators.Location(latitude, longitude), decorators.Name(ws.Station.Name))

			entity, err := entities.New(entityID, fiware.WeatherObservedTypeName, attributes...)
			if err != nil {
				log.Error().Err(err).Msg("failed to construct new entity")
				return err
			}

			_, err = i.cb.CreateEntity(ctx, entity, headers)
			if err != nil {
				log.Error().Err(err).Msg("failed to create entity")
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
			return nil, fmt.Errorf("failed to parse temperature from string: %s", err.Error())
		}

		layout := "2006-01-02 15:04:05"
		t, err := time.Parse(layout, ws.Logg[0].DateTime)
		if err != nil {
			return nil, fmt.Errorf("failed to parse time from string: %s", err.Error())
		}

		utcTime := t.UTC().Format(time.RFC3339)

		attributes := append(
			make([]entities.EntityDecoratorFunc, 0, 2),
			decorators.Number("temperature", temp, properties.ObservedAt(utcTime)),
			decorators.DateTime("dateObserved", utcTime),
		)

		return attributes, nil
	} else {
		json.Marshal(ws)
		return nil, fmt.Errorf("weather station response does not contain loggs: %s", ws)
	}
}
