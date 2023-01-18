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
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type IntegrationFimbul interface {
	CreateWeatherObserved(ctx context.Context, stationIds func() []StationID) error
}

type StationID string

type intFim struct {
	cb      client.ContextBrokerClient
	service string
}

func New(cb client.ContextBrokerClient, service string) IntegrationFimbul {
	return &intFim{
		cb:      cb,
		service: service,
	}
}

func (i intFim) CreateWeatherObserved(ctx context.Context, stationIds func() []StationID) error {
	logging.GetFromContext(ctx) //use this log instead of fmt.Errorf
	client := http.Client{}

	for _, id := range stationIds() {
		url := fmt.Sprintf("%s/stations/%s.last", i.service, id)

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %s", err.Error())
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to send request: %s", err.Error())
		}

		if resp.StatusCode != 200 {
			return fmt.Errorf("request failed expected status code 200 but got: %d", resp.StatusCode)
		}

		ws := struct {
			Station weatherStation `json:"station"`
		}{}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %s", err.Error())
		}

		err = json.Unmarshal(bodyBytes, &ws)
		if err != nil {
			return fmt.Errorf("failed to unmarshal response body into json: %s", err.Error())
		}

		//TODO: should have some prefix later
		entityID := fiware.WeatherObservedIDPrefix + ws.Station.ID

		attributes, err := createWeatherObservedAttributes(ctx, ws.Station)
		if err != nil {
			return fmt.Errorf("failed to create attributes for entity: %s", err.Error())
		}

		fragment, _ := entities.NewFragment(attributes...)

		headers := map[string][]string{"Content-Type": {"application/ld+json"}}

		_, err = i.cb.MergeEntity(ctx, entityID, fragment, headers)
		if err != nil {
			if !errors.Is(err, ngsierrors.ErrNotFound) {
				return fmt.Errorf("failed to merge entity: %s", err.Error())
			}

			latitude, err := strconv.ParseFloat(ws.Station.Latitude, 64)
			if err != nil {
				return fmt.Errorf("failed to parse latitude from string: %s", err.Error())
			}
			longitude, err := strconv.ParseFloat(ws.Station.Longitude, 64)
			if err != nil {
				return fmt.Errorf("failed to parse longitude from string: %s", err.Error())
			}

			attributes = append(attributes, decorators.Location(latitude, longitude), decorators.Name(ws.Station.Name))

			entity, err := entities.New(entityID, fiware.WeatherObservedTypeName, attributes...)
			if err != nil {
				return fmt.Errorf("failed to construct new entity: %s", err.Error())
			}

			_, err = i.cb.CreateEntity(ctx, entity, headers)
			if err != nil {
				return fmt.Errorf("failed to create entity: %s", err.Error())
			}
		}
	}

	return nil
}

func createWeatherObservedAttributes(ctx context.Context, ws weatherStation) ([]entities.EntityDecoratorFunc, error) {
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
		decorators.Temperature(temp),
		decorators.DateTime("dateObserved", utcTime),
	)

	return attributes, nil
}
