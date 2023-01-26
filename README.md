# integration-fimbul
A service that integrates data between fimbul.se and our City Information Platform


# running locally
Build image from Dockerfile:

    `docker build -f deployments/Dockerfile -t yourtag:version .`

Or use go run:

    `go run ./cmd/integration-fimbul/main.go`

# environment variables
Make sure to set all environment variables before running.

| Variable | Description |
| ---------|-------------|
|FIMBUL_URL | url to the service we are retrieving data from.|
|WO_PREFIX | added to the end of the standard fiware prefix for a WeatherObserved entity ("urn:ngsi-ld:WeatherObserved:"). A suggestion is to add a prefix that suggests locale or ownership of the entity measured, such as *se:sundsvall:* or *se:diwise:*. |
|CONTEXT_BROKER_URL | url to the context broker that the data is being sent to.|
|stations | set as a string flag with the name "stations", and separate station IDs in the string by comma. e.g. *-stations="station1,station2,station3"*.|