package application

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/diwise/context-broker/pkg/ngsild"
	ngsierrors "github.com/diwise/context-broker/pkg/ngsild/errors"
	"github.com/diwise/context-broker/pkg/ngsild/types"
	test "github.com/diwise/context-broker/pkg/test"
	testhttp "github.com/diwise/service-chassis/pkg/test/http"
	"github.com/diwise/service-chassis/pkg/test/http/response"
	"github.com/matryer/is"
)

func TestGetCurrentWeather(t *testing.T) {
	is, ctxBroker, service := testSetup(t)

	id := []StationID{"S-vall-01-02"}
	app := New(ctxBroker, service.URL())
	err := app.CreateWeatherObserved(context.Background(), func() []StationID {
		return id
	})
	is.NoErr(err)
	is.Equal(len(ctxBroker.MergeEntityCalls()), 1)
	is.Equal(len(ctxBroker.CreateEntityCalls()), 1)
}

func TestGetTimeParsedCorrectly(t *testing.T) {
	is, ctxBroker, service := testSetup(t)

	id := []StationID{"S-vall-01-02"}
	app := New(ctxBroker, service.URL())
	err := app.CreateWeatherObserved(context.Background(), func() []StationID {
		return id
	})
	is.NoErr(err)
	is.Equal(len(ctxBroker.MergeEntityCalls()), 1)
	is.Equal(len(ctxBroker.CreateEntityCalls()), 1)

	entity := ctxBroker.MergeEntityCalls()[0].Fragment

	entityBytes, err := json.Marshal(entity)
	is.NoErr(err)

	fmt.Println(string(entityBytes))

	dateObserved := `"dateObserved":{"type":"Property","value":{"@type":"DateTime","@value":"2023-01-13T15:40:00Z"}}`

	is.True(strings.Contains(string(entityBytes), dateObserved))
}

func testSetup(t *testing.T) (*is.I, *test.ContextBrokerClientMock, testhttp.MockService) {
	is := is.New(t)
	ctxBroker := &test.ContextBrokerClientMock{
		MergeEntityFunc: func(ctx context.Context, entityID string, fragment types.EntityFragment, headers map[string][]string) (*ngsild.MergeEntityResult, error) {
			return nil, ngsierrors.ErrNotFound
		},
		CreateEntityFunc: func(ctx context.Context, entity types.Entity, headers map[string][]string) (*ngsild.CreateEntityResult, error) {
			return nil, nil
		},
	}

	service := testhttp.NewMockServiceThat(
		testhttp.Expects(is),
		testhttp.Returns(response.Code(http.StatusOK), response.Body([]byte(testData))),
	)

	return is, ctxBroker, service
}

const testData string = `{"station":{
    "STATION_ID": "S-vall-01-02",
    "NAME": "Sundsvall Södra berget",
    "CUSTOMER": "Sundsvall",
    "LAT": "62.36623300",
    "LON": "17.30874500",
    "ELEVATION": "",
    "logg":[{
		"MESSAGE_DATE_TIME": "2023-01-13 15:40:00",
		"WIND_MINIMUM_SPEED": "1.1",
		"WIND_AVERAGE_SPEED": "1.9",
		"WIND_MAXIMUM_SPEED": "3.1",
		"WIND_DIRECTION": "62.0",
		"WIND_DIRECTION_VARIABILITY": "5.0",
		"TEMPERATURE": "-1.0",
		"RELATIVE_HUMIDITY": "100.0"
    }]
}
} `
