package servicob

import "math"

type Temperatura struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func NovaTemperatura(cidade string, celsius float64) Temperatura {
	return Temperatura{
		City:  cidade,
		TempC: arredonda(celsius),
		TempF: arredonda(celsius*1.8 + 32),
		TempK: arredonda(celsius + 273),
	}
}

func arredonda(valor float64) float64 {
	return math.Round(valor*100) / 100
}
