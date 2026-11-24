package servicob

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var ErrCepNaoEncontrado = errors.New("can not find zipcode")

type ViaCEP struct {
	Cep         string `json:"cep"`
	Logradouro  string `json:"logradouro"`
	Complemento string `json:"complemento"`
	Bairro      string `json:"bairro"`
	Localidade  string `json:"localidade"`
	Uf          string `json:"uf"`
	Estado      string `json:"estado"`
	Ibge        string `json:"ibge"`
}

type ViaCEPClient struct {
	BaseURL string
	Client  *http.Client
}

func (v *ViaCEPClient) BuscaCep(ctx context.Context, cep string) (*ViaCEP, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.BaseURL+"/ws/"+cep+"/json/", nil)
	if err != nil {
		return nil, err
	}

	resp, err := v.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		return nil, ErrCepNaoEncontrado
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("viacep respondeu com status %d", resp.StatusCode)
	}

	var c ViaCEP
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return nil, err
	}
	if c.Localidade == "" {
		return nil, ErrCepNaoEncontrado
	}
	return &c, nil
}
