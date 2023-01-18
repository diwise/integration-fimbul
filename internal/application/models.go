package application

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
