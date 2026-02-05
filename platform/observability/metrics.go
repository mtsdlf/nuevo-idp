package observability

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total de requests HTTP por servicio, entorno, método, ruta y status.",
		},
		[]string{"service", "env", "method", "route", "status"},
	)

	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duración de requests HTTP por servicio, entorno, método y ruta.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "env", "method", "route"},
	)

	domainEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "domain_events_total",
			Help: "Total de eventos de dominio por tipo y resultado.",
		},
		[]string{"event", "result"},
	)

	downstreamErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "downstream_errors_total",
			Help: "Total de errores al llamar a servicios downstream por destino, código y status.",
		},
		[]string{"target", "code", "status"},
	)

	workflowRunDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "workflow_run_duration_seconds",
			Help:    "Duración de ejecuciones de workflows por nombre y resultado.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"workflow", "result"},
	)

	workflowRetriesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "workflow_retries_total",
			Help: "Total de reintentos de workflows por nombre.",
		},
		[]string{"workflow"},
	)
)

// InitMetrics registra los collectors HTTP globales. Debe llamarse una vez en main.
func InitMetrics() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDurationSeconds, domainEventsTotal, downstreamErrorsTotal, workflowRunDurationSeconds, workflowRetriesTotal)
}

// ObserveDomainEvent incrementa un contador para eventos de dominio de alto nivel.
// "event" debería ser un nombre estable de caso de uso (por ejemplo, "application_created").
// "result" típicamente será "success" o "error".
func ObserveDomainEvent(event, result string) {
	domainEventsTotal.WithLabelValues(event, result).Inc()
}

// ObserveDownstreamError incrementa un contador para errores al llamar a
// servicios externos (por ejemplo, control-plane-api, execution-workers).
// "target" es el nombre lógico del servicio; "code" suele provenir de un
// error tipado del dominio y "status" es el HTTP status code.
func ObserveDownstreamError(target, code string, status int) {
	statusStr := strconv.Itoa(status)
	if code == "" {
		code = "unknown_error"
	}
	downstreamErrorsTotal.WithLabelValues(target, code, statusStr).Inc()
}

// ObserveWorkflowDuration registra la duración de una ejecución de workflow
// en segundos, etiquetada por nombre lógico y resultado (success/error).
func ObserveWorkflowDuration(workflowName, result string, seconds float64) {
	if workflowName == "" {
		workflowName = "unknown"
	}
	if result == "" {
		result = "unknown"
	}
	workflowRunDurationSeconds.WithLabelValues(workflowName, result).Observe(seconds)
}

// ObserveWorkflowRetries incrementa el contador de reintentos de un workflow
// por el número de reintentos observados (por ejemplo, Attempt-1 de Temporal).
func ObserveWorkflowRetries(workflowName string, retries int) {
	if workflowName == "" {
		workflowName = "unknown"
	}
	if retries <= 0 {
		return
	}
	workflowRetriesTotal.WithLabelValues(workflowName).Add(float64(retries))
}

// InstrumentHTTP envuelve un handler para medir cantidad y duración de requests.
func InstrumentHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		service := os.Getenv("SERVICE_NAME")
		if service == "" {
			service = "unknown-service"
		}
		env := os.Getenv("ENVIRONMENT")
		if env == "" {
			env = "unknown"
		}

		path := r.URL.Path
		route := normalizeRoute(path)
		method := r.Method
		status := strconv.Itoa(rec.status)

		httpRequestsTotal.WithLabelValues(service, env, method, route, status).Inc()
		httpRequestDurationSeconds.WithLabelValues(service, env, method, route).Observe(time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// normalizeRoute intenta reducir la cardinalidad de las rutas HTTP
// reemplazando IDs numéricos o UUIDs comunes por comodines.
func normalizeRoute(path string) string {
	if path == "" {
		return "/"
	}
	segments := strings.Split(path, "/")
	for i, s := range segments {
		if s == "" {
			continue
		}
		// Reemplazo simple de segmentos que parecen IDs numéricos.
		if isNumeric(s) || looksLikeUUID(s) {
			segments[i] = "{id}"
		}
	}
	return strings.Join(segments, "/")
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

func looksLikeUUID(s string) bool {
	// Heurística mínima para evitar depender de paquetes extra.
	if len(s) != 36 {
		return false
	}
	return strings.Count(s, "-") == 4
}
