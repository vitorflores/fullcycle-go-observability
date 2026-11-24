package cep

import (
	"errors"
	"strings"
	"testing"
)

func TestLerEntrada(t *testing.T) {
	casos := []struct {
		nome   string
		body   string
		cep    string
		valido bool
	}{
		{"cep com 8 digitos", `{"cep": "29902555"}`, "29902555", true},
		{"cep com 7 digitos", `{"cep": "2990255"}`, "", false},
		{"cep com 9 digitos", `{"cep": "299025555"}`, "", false},
		{"cep com hifen", `{"cep": "29902-555"}`, "", false},
		{"cep com letras", `{"cep": "2990255a"}`, "", false},
		{"cep como numero", `{"cep": 29902555}`, "", false},
		{"cep ausente", `{}`, "", false},
		{"json quebrado", `{"cep": `, "", false},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			cep, err := LerEntrada(strings.NewReader(c.body))
			if c.valido {
				if err != nil {
					t.Fatalf("esperava sucesso, recebi erro %v", err)
				}
				if cep != c.cep {
					t.Fatalf("esperava cep %q, recebi %q", c.cep, cep)
				}
				return
			}
			if !errors.Is(err, ErrCepInvalido) {
				t.Fatalf("esperava ErrCepInvalido, recebi %v", err)
			}
		})
	}
}
