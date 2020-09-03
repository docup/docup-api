package main

import (
	"context"
	"fmt"
	"log"
	stdlog "log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/docup/docup-api/env"
	"github.com/docup/docup-api/graph"
	"github.com/docup/docup-api/graph/generated"
	"github.com/docup/docup-api/log2"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const defaultPort = "8080"

func main() {
	ctx := context.Background()

	// Env
	env, err := env.Process()
	if err != nil {
		stdlog.Fatalf(err.Error())
	}

	logger, err := log2.New(env.LogLevel, "docup-api", env.Env)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "[ERROR] Failed to setup logger: %s\n", err)
		log.Fatal(err)
	}
	defer func() {
		_ = logger.Sync()
	}()
	ctx = log2.WithContext(ctx, logger)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	})

	tokenVerifier, err := NewFirebaseAuthTokenVerifier(ctx, "docup-269111")
	if err != nil {
		stdlog.Fatalf(err.Error())
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	resolver := &graph.Resolver{}

	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: resolver}))
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * time.Second,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	})
	//srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	//	res := next(ctx)
	//	return res
	//})
	//srv.AroundFields(func(ctx context.Context, next graphql.Resolver) (res interface{}, err error) {
	//	res, err = next(ctx)
	//	return res, err
	//})
	//srv.AroundResponses(func(ctx context.Context, next graphql.ResponseHandler) *graphql.Response {
	//	res := next(ctx)
	//	return res
	//})

	r := chi.NewRouter()

	// protected routes
	r.Group(func(r chi.Router) {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token, err := tokenVerifier.Verify(ctx, r)
				if err != nil {
					http.Error(w, http.StatusText(401), 401)
					w.Write([]byte(err.Error()))
					return
				}
				newContext := context.WithValue(r.Context(), "GoogleIDToken", token)
				next.ServeHTTP(w, r.WithContext(newContext))
			})
		})

		r.Handle("/private-query", c.Handler(srv))
	})

	// public routes
	r.Group(func(r chi.Router) {
		r.Handle("/", playground.Handler("GraphQL playground", "/query"))
		r.Handle("/query", c.Handler(srv))
	})

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
