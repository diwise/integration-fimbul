package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
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
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
)

/*Skapa ett enkelt program som gör ett http-anrop mot sensorn och skapar en WeatherObserved per sensor. Sensorerna skall kunna skickas in med ett kommandoradsargument.*/

type wsLog struct {
	DateTime                 string `json:"MESSAGE_DATE_TIME"`
	WindMinimumSpeed         string `json:"WIND_MINIMUM_SPEED"`
	WindAverageSpeed         string `json:"WIND_AVERAGE_SPEED"`
	WindMaximumSpeed         string `json:"WIND_MAXIMUM_SPEED"`
	WindDirection            string `json:"WIND_DIRECTION"`
	WindDirectionVariability string `json:"WIND_DIRECTION_VARIABILITY"`
	Temperature              string `json:"TEMPERATURE"`
	RelativeHumidity         string `json:"RELATIVE_HUMIDITY"`
}

type weatherStation struct {
	ID        string  `json:"STATION_ID"`
	Name      string  `json:"NAME"`
	Customer  string  `json:"CUSTOMER"`
	Latitude  string  `json:"LAT"`
	Longitude string  `json:"LON"`
	Elevation string  `json:"ELEVATION"`
	Logg      []wsLog `json:"logg"`
}

var stationId string

func main() {
	ctx, log, cleanup := o11y.Init(context.Background(), "integration-fimbul", "")
	defer cleanup()

	service := env.GetVariableOrDie(log, "FIMBUL_URL", "url to weather service")
	flag.StringVar(&stationId, "stationId", "", "id of the station to retrieve data from")
	flag.Parse()

	ctxBrokerURL := env.GetVariableOrDie(log, "CONTEXT_BROKER_URL", "url to context broker")
	ctxBroker := client.NewContextBrokerClient(ctxBrokerURL, client.Debug("true"))

	getCurrentWeatherFromWeatherStation(ctx, service, stationId, ctxBroker)

}

func getCurrentWeatherFromWeatherStation(ctx context.Context, service, id string, ctxBroker client.ContextBrokerClient) error {
	client := http.Client{}

	url := service + "/stations/" + id + ".last" //fmt.sprintf

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %s", err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		return err //fix in a bit
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("request failed expected status code 200 but got: %d", resp.StatusCode)
	}

	ws := struct {
		Station weatherStation `json:"station"`
	}{}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bodyBytes, &ws)
	if err != nil {
		return err
	}

	entityID := fiware.WeatherObservedIDPrefix + ws.Station.ID //should have some prefix later

	attributes, err := createWeatherObservedAttributes(ctx, ws.Station)
	if err != nil {
		return err
	}

	fragment, _ := entities.NewFragment(attributes...)

	headers := map[string][]string{"Content-Type": {"application/ld+json"}}

	_, err = ctxBroker.MergeEntity(ctx, entityID, fragment, headers)
	if err != nil {
		if !errors.Is(err, ngsierrors.ErrNotFound) {
			return err
		}

		latitude, err := strconv.ParseFloat(ws.Station.Latitude, 64)
		if err != nil {
			return err
		}
		longitude, err := strconv.ParseFloat(ws.Station.Longitude, 64)
		if err != nil {
			return err
		}

		attributes = append(attributes, decorators.Location(latitude, longitude), decorators.Name(ws.Station.Name))

		entity, err := entities.New(entityID, fiware.WeatherObservedTypeName, attributes...)
		if err != nil {
			return err
		}

		_, err = ctxBroker.CreateEntity(ctx, entity, headers)
		if err != nil {
			return fmt.Errorf("failed to create entity: %s", err.Error())
		}

		return err //fill in soon
	}

	return nil
}

func createWeatherObservedAttributes(ctx context.Context, ws weatherStation) ([]entities.EntityDecoratorFunc, error) {
	temp, err := strconv.ParseFloat(ws.Logg[0].Temperature, 64)
	if err != nil {
		return nil, err
	}

	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, ws.Logg[0].DateTime)
	if err != nil {
		return nil, err
	}

	utcTime := t.UTC().Format(time.RFC3339)

	attributes := append(
		make([]entities.EntityDecoratorFunc, 0, 2),
		decorators.Temperature(temp),
		decorators.DateTime("dateObserved", utcTime),
	)

	return attributes, nil
}
