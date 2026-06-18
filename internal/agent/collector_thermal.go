package agent

import (
	"os"
	"strconv"
	"strings"

	"github.com/michal/kiviq/internal/shared"
)

func collectThermals() []shared.TempInfo {
	var temps []shared.TempInfo

	for i := 0; ; i++ {
		basePath := "/sys/class/thermal/thermal_zone" + strconv.Itoa(i)

		typePath := basePath + "/type"
		typeData, err := os.ReadFile(typePath)
		if err != nil {
			break
		}
		label := strings.TrimSpace(string(typeData))

		tempPath := basePath + "/temp"
		tempData, err := os.ReadFile(tempPath)
		if err != nil {
			continue
		}
		tempMilli, err := strconv.ParseFloat(strings.TrimSpace(string(tempData)), 64)
		if err != nil {
			continue
		}

		temp := shared.TempInfo{
			Label:   label,
			Celsius: tempMilli / 1000,
		}

		tripPath := basePath + "/trip_point_0_temp"
		if tripData, err := os.ReadFile(tripPath); err == nil {
			if tripVal, err := strconv.ParseFloat(strings.TrimSpace(string(tripData)), 64); err == nil {
				temp.HighCelsius = tripVal / 1000
			}
		}

		temps = append(temps, temp)
	}

	return temps
}
