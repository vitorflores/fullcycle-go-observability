package servicob

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/vitorflores/fullcycle-go-observability/internal/cep"
)

type Config struct {
	RequestNameOTEL string
	OTELTracer      trace.Tracer
	ViaCEP          *ViaCEPClient
	Clima           *ClimaClient
}

type Webserver struct {
	Config *Config
}

func NewServer(config *Config) *Webserver {
	return &Webserver{
		Config: config,
	}
}

func (we *Webserver) CreateServer() http.Handler {
	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Post("/", we.HandleRequest)

	return otelhttp.NewHandler(router, "servico-b")
}

func (h *Webserver) HandleRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.Config.OTELTracer.Start(r.Context(), h.Config.RequestNameOTEL)
	defer span.End()

	codigo, err := cep.LerEntrada(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	ctxCep, spanCep := h.Config.OTELTracer.Start(ctx, "busca-cep")
	spanCep.SetAttributes(attribute.String("cep", codigo))
	endereco, err := h.Config.ViaCEP.BuscaCep(ctxCep, codigo)
	if err != nil {
		spanCep.RecordError(err)
		spanCep.SetStatus(codes.Error, err.Error())
	}
	spanCep.End()

	if errors.Is(err, ErrCepNaoEncontrado) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctxClima, spanClima := h.Config.OTELTracer.Start(ctx, "busca-temperatura")
	spanClima.SetAttributes(attribute.String("cidade", endereco.Localidade))
	celsius, err := h.Config.Clima.BuscaTemperatura(ctxClima, endereco.Localidade, endereco.Estado)
	if err != nil {
		spanClima.RecordError(err)
		spanClima.SetStatus(codes.Error, err.Error())
	}
	spanClima.End()

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(NovaTemperatura(endereco.Localidade, celsius))
}
