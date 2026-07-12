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
	"AUH": "Asia/Dubai",
	"SHJ": "Asia/Dubai",
	"BOM": "Asia/Kolkata",
	"BLR": "Asia/Kolkata",
	"CCU": "Asia/Kolkata",
	"CAN": "Asia/Shanghai",
	"PEK": "Asia/Shanghai",
	"BKK": "Asia/Bangkok",
	"KUL": "Asia/Kuala_Lumpur",
	"SIN": "Asia/Singapore",
	"HKG": "Asia/Hong_Kong",
	"KWI": "Asia/Kuwait",
	"DMM": "Asia/Riyadh",
	"RUH": "Asia/Riyadh",
	"CMB": "Asia/Colombo",
	"DAC": "Asia/Dhaka",
	"LXA": "Asia/Shanghai",
	"KMG": "Asia/Shanghai",
	"SZX": "Asia/Shanghai",
	"DMK": "Asia/Bangkok",
	"ICN": "Asia/Seoul",
	"NRT": "Asia/Tokyo",
	"ALA": "Asia/Almaty",
	"TAS": "Asia/Tashkent",
	"DYU": "Asia/Dushanbe",
	"FRU": "Asia/Bishkek",
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
