# integration-fimbul
A service that integrates data between fimbul.se and our City Information Platform


# running locally
Build image from Dockerfile or use go run

Make sure to set all environment variables before running.

FIMBUL_URL - url to the service we are retrieving data from.

WO_PREFIX - added to the end of the standard fiware prefix for a WeatherObserved entity ("urn:ngsi-ld:WeatherObserved:"). A suggestion is to add a prefix that suggest locale or ownership of the entity measured. 

CONTEXT_BROKER_URL - url to the context broker that the data is being sent to.

The ID(s) of the station(s) you want to retrieve data from should be set in the environment as a string flag with the name "stations". Separate station IDs in the string by comma, i.e. "station1,station2,station3"