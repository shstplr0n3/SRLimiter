package examples

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"srlimiter.go"
)

type RateMiddleware struct {
	collector *srlimiter.Collector
	limiter   *someRateLimiter
}

func NewRateMiddleware() *RateMiddleware {
	return &RateMiddleware{
		collector: srlimiter.NewCollector(),
	}
}

func getPriorityFromRequest(r *http.Request) uint16 {
	switch r.URL.Path {
	case "/service1/api/v1/resource1":
		return 100
	case "/service1/api/v1/resource2":
		return 50
	case "/service2/api/v2/resource1":
		return 25
	default:
		return 10
	}
}

func (rm *RateMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		priority := getPriorityFromRequest(r)
		load := srlimiter.NewLoad(r, priority)
		rm.collector.AddLoad(load)

		processedReq := rm.collector.GetNextLoad()
		if processedReq != nil {
			request := processedReq.GetProcess().(*http.Request)
			next.ServeHTTP(w, request)
		}
	})
}

func main() {
	megaRouter := chi.NewRouter()

	rateMiddleware1 := NewRateMiddleware()
	rateMiddleware2 := NewRateMiddleware()

	router1 := chi.NewRouter()
	router1.Use(rateMiddleware1.Handle)
	router1.Get("/api/v1/resource1", handler1)
	router1.Get("/api/v1/resource2", handler2)

	router2 := chi.NewRouter()
	router2.Use(rateMiddleware2.Handle)
	router2.Get("/api/v2/resource1", handler3)

	megaRouter.Mount("/service1", router1)
	megaRouter.Mount("/service2", router2)

	http.ListenAndServe(":8080", megaRouter)
}
