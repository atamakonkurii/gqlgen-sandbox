package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/osuosu/gqlgen-todos/graph"
	"github.com/rs/cors"
	"golang.org/x/exp/slices"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	router := chi.NewRouter()

	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8080", "http://localhost:8000"},
		AllowCredentials: true,
		Debug:            true,
	}).Handler)

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))

	srv.AddTransport(transport.Websocket{
		// Keep-alives are important for WebSockets to detect dead connections. This is
		// not unlike asking a partner who seems to have zoned out while you tell them
		// a story crucial to understanding the dynamics of your workplace: "Are you
		// listening to me?"
		//
		// Failing to set a keep-alive interval can result in the connection being held
		// open and the server expending resources to communicate with a client that has
		// long since walked to the kitchen to make a sandwich instead.
		KeepAlivePingInterval: 10 * time.Second,

		// The `github.com/gorilla/websocket.Upgrader` is used to handle the transition
		// from an HTTP connection to a WebSocket connection. Among other options, here
		// you must check the origin of the request to prevent cross-site request forgery
		// attacks.
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow exact match on host.
				origin := r.Header.Get("Origin")
				if origin == "" || origin == r.Header.Get("Host") {
					return true
				}

				// Match on allow-listed origins.
				return slices.Contains([]string{"http://localhost:8080", "http://localhost:8000"}, origin)
			},
		},
	})
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	router.Handle("/", playground.Handler("GraphQL playground", "/graphql"))
	router.Handle("/graphql", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
