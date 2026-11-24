package servicoa

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	"github.com/vitorflores/fullcycle-go-observability/internal/cep"
)

type Config struct {
	ServicoBURL     string
	RequestNameOTEL string
	OTELTracer      trace.Tracer
}

type Webserver struct {
	Config *Config
	client *http.Client
}

func NewServer(config *Config) *Webserver {
	return &Webserver{
		Config: config,
		client: &http.Client{Timeout: 30 * time.Second},
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

	return otelhttp.NewHandler(router, "servico-a")
}

func (h *Webserver) HandleRequest(w http.ResponseWriter, r *http.Request) {
	ctx, span := h.Config.OTELTracer.Start(r.Context(), h.Config.RequestNameOTEL)
	defer span.End()

	codigo, err := cep.LerEntrada(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	payload, err := json.Marshal(cep.Entrada{Cep: codigo})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.Config.ServicoBURL, bytes.NewReader(payload))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := h.client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	w.Write(bodyBytes)
}
