package cep

import (
	"encoding/json"
	"errors"
	"io"
	"regexp"
)

var ErrCepInvalido = errors.New("invalid zipcode")

var formatoCep = regexp.MustCompile(`^\d{8}$`)

type Entrada struct {
	Cep string `json:"cep"`
}

func Valido(cep string) bool {
	return formatoCep.MatchString(cep)
}

func LerEntrada(body io.Reader) (string, error) {
	var entrada Entrada
	if err := json.NewDecoder(body).Decode(&entrada); err != nil {
		return "", ErrCepInvalido
	}
	if !Valido(entrada.Cep) {
		return "", ErrCepInvalido
	}
	return entrada.Cep, nil
}
