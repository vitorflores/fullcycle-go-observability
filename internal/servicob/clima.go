package servicob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var ErrCidadeNaoEncontrada = errors.New("cidade não encontrada na base de clima")

type geocodingResposta struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Admin1    string  `json:"admin1"`
	} `json:"results"`
}

type previsaoResposta struct {
	Current struct {
		Temperature2m float64 `json:"temperature_2m"`
	} `json:"current"`
}

type ClimaClient struct {
	GeocodingURL string
	PrevisaoURL  string
	Client       *http.Client
}

func (c *ClimaClient) BuscaTemperatura(ctx context.Context, cidade, estado string) (float64, error) {
	lat, lon, err := c.buscaCoordenadas(ctx, cidade, estado)
	if err != nil {
		return 0, err
	}

	params := url.Values{}
	params.Set("latitude", strconv.FormatFloat(lat, 'f', -1, 64))
	params.Set("longitude", strconv.FormatFloat(lon, 'f', -1, 64))
	params.Set("current", "temperature_2m")

	var previsao previsaoResposta
	if err := c.getJSON(ctx, c.PrevisaoURL+"/v1/forecast?"+params.Encode(), &previsao); err != nil {
		return 0, err
	}
	return previsao.Current.Temperature2m, nil
}

func (c *ClimaClient) buscaCoordenadas(ctx context.Context, cidade, estado string) (float64, float64, error) {
	params := url.Values{}
	params.Set("name", cidade)
	params.Set("count", "10")
	params.Set("language", "pt")
	params.Set("countryCode", "BR")

	var geo geocodingResposta
	if err := c.getJSON(ctx, c.GeocodingURL+"/v1/search?"+params.Encode(), &geo); err != nil {
		return 0, 0, err
	}
	if len(geo.Results) == 0 {
		return 0, 0, ErrCidadeNaoEncontrada
	}

	for _, r := range geo.Results {
		if estado != "" && r.Admin1 == estado {
			return r.Latitude, r.Longitude, nil
		}
	}
	return geo.Results[0].Latitude, geo.Results[0].Longitude, nil
}

func (c *ClimaClient) getJSON(ctx context.Context, endereco string, destino any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endereco, nil)
	if err != nil {
		return err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("open-meteo respondeu com status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(destino)
}
