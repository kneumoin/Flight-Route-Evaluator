package scoring

import (
	"fmt"
	"time"
)

var airportTZ = map[string]string{
	"MOW": "Europe/Moscow",
	"KTM": "Asia/Kathmandu",
	"DXB": "Asia/Dubai",
	"DOH": "Asia/Qatar",
	"DEL": "Asia/Kolkata",
	"IST": "Europe/Istanbul",
	"TFU": "Asia/Shanghai",
}

func AirportLocation(iata string) (*time.Location, error) {
	tz, ok := airportTZ[iata]
	if !ok {
		return nil, fmt.Errorf("unknown airport IATA: %s", iata)
	}
	return time.LoadLocation(tz)
}

func ConnectionDuration(arrival, departure time.Time) time.Duration {
	return departure.UTC().Sub(arrival.UTC())
}
