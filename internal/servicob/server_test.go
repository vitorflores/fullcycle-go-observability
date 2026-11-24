package servicob

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel/trace/noop"
)

func novoServidorFake(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/01001000/json/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"cep":"01001-000","localidade":"São Paulo","uf":"SP","estado":"São Paulo"}`))
	})
	mux.HandleFunc("/ws/99999999/json/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"erro":"true"}`))
	})
	mux.HandleFunc("/v1/search", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[
			{"name":"São Paulo","latitude":-9.45,"longitude":-36.03,"admin1":"Alagoas"},
			{"name":"São Paulo","latitude":-23.54,"longitude":-46.63,"admin1":"São Paulo"}
		]}`))
	})
	mux.HandleFunc("/v1/forecast", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("latitude") != "-23.54" {
			t.Errorf("latitude errada na previsão: %s", r.URL.Query().Get("latitude"))
		}
		w.Write([]byte(`{"current":{"temperature_2m":28.5}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func novoHandler(baseURL string) http.Handler {
	config := &Config{
		RequestNameOTEL: "teste",
		OTELTracer:      noop.NewTracerProvider().Tracer("teste"),
		ViaCEP:          &ViaCEPClient{BaseURL: baseURL, Client: http.DefaultClient},
		Clima:           &ClimaClient{GeocodingURL: baseURL, PrevisaoURL: baseURL, Client: http.DefaultClient},
	}
	return NewServer(config).CreateServer()
}

func TestHandleRequest(t *testing.T) {
	handler := novoHandler(novoServidorFake(t).URL)

	casos := []struct {
		nome   string
		body   string
		status int
		corpo  string
	}{
		{"cep invalido", `{"cep": "123"}`, http.StatusUnprocessableEntity, "invalid zipcode"},
		{"cep inexistente", `{"cep": "99999999"}`, http.StatusNotFound, "can not find zipcode"},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(c.body)))
			if rec.Code != c.status {
				t.Fatalf("esperava status %d, recebi %d", c.status, rec.Code)
			}
			if strings.TrimSpace(rec.Body.String()) != c.corpo {
				t.Fatalf("esperava corpo %q, recebi %q", c.corpo, rec.Body.String())
			}
		})
	}

	t.Run("cep valido", func(t *testing.T) {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"cep": "01001000"}`)))
		if rec.Code != http.StatusOK {
			t.Fatalf("esperava status 200, recebi %d: %s", rec.Code, rec.Body.String())
		}
		var resposta Temperatura
		if err := json.NewDecoder(rec.Body).Decode(&resposta); err != nil {
			t.Fatal(err)
		}
		esperado := Temperatura{City: "São Paulo", TempC: 28.5, TempF: 83.3, TempK: 301.5}
		if resposta != esperado {
			t.Fatalf("esperava %+v, recebi %+v", esperado, resposta)
		}
	})
}
