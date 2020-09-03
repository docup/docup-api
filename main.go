package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/docup/docup-api/env"
	"github.com/docup/docup-api/graph"
	"github.com/docup/docup-api/graph/generated"
	"github.com/docup/docup-api/log2"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/openshift/osin"
	"github.com/rs/cors"
)

const defaultPort = "8080"

func main() {
	ctx := context.Background()

	// Env
	env, err := env.Process()
	if err != nil {
		log.Fatalf(err.Error())
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

	tokenVerifier, err := NewFirebaseAuthTokenVerifier(ctx, env.ProjectID)
	if err != nil {
		log.Fatalf(err.Error())
	}

	// Use the application default credentials
	conf := &firebase.Config{ProjectID: env.ProjectID}
	app, err := firebase.NewApp(ctx, conf)
	if err != nil {
		log.Fatalln(err)
	}

	firestoreClient, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer firestoreClient.Close()

	// OAuth server
	storage := &OsinStorage{
		Firestore: firestoreClient,
	}
	OAuthServer := osin.NewServer(osin.NewServerConfig(), storage)

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	resolver := &graph.Resolver{
		Firestore: firestoreClient,
	}

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
		// sample request
		// http://localhost:8080/authorize?redirect_urihttp%3A%2F%2Flocalhost%3A8080%2Fredirecturi&response_type=code&client_id=id&state=1
		r.Get("/authorize", func(w http.ResponseWriter, r *http.Request) {
			resp := OAuthServer.NewResponse()
			defer resp.Close()
			if ar := OAuthServer.HandleAuthorizeRequest(resp, r); ar != nil {
				//id := r.FormValue("id")
				//password := r.FormValue("password")

				//csrftoken checkなど
				//...

				//DBにUserとPassword確認
				//err := a.DBClient.CheckIDPassword(id, password)
				//if err == sql.ErrNoRows {
				//
				//	// NoRows場合の処理
				//
				//} else if err != nil {
				//
				//	// error処理
				//
				//}
				ar.Authorized = true
				OAuthServer.FinishAuthorizeRequest(resp, r, ar)
				osin.OutputJSON(resp, w, r)
			} else {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("bad request"))
			}
		})

		r.Handle("/", playground.Handler("GraphQL playground", "/query"))
		r.Handle("/query", c.Handler(srv))
	})

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe("localhost:"+port, r))
}

type App struct {
	OAuthServer *osin.Server
}

type OsinStorage struct {
	Firestore *firestore.Client
}

func (it *OsinStorage) Clone() osin.Storage {
	return it
}

func (OsinStorage) Close() {
	// do nothing
}

func (OsinStorage) GetClient(id string) (osin.Client, error) {
	return OsinClient{}, nil
}

func (OsinStorage) SaveAuthorize(d *osin.AuthorizeData) error {
	return nil
}

func (OsinStorage) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	panic("5 implement me")
}

func (OsinStorage) RemoveAuthorize(code string) error {
	panic("6 implement me")
}

func (OsinStorage) SaveAccess(*osin.AccessData) error {
	panic("7 implement me")
}

func (OsinStorage) LoadAccess(token string) (*osin.AccessData, error) {
	panic("8 implement me")
}

func (OsinStorage) RemoveAccess(token string) error {
	panic("9 implement me")
}

func (OsinStorage) LoadRefresh(token string) (*osin.AccessData, error) {
	panic("10 implement me")
}

func (OsinStorage) RemoveRefresh(token string) error {
	panic("11 implement me")
}

type OsinClient struct{}

func (OsinClient) GetId() string {
	return "id"
}

func (OsinClient) GetSecret() string {
	return "secret"
}

func (OsinClient) GetRedirectUri() string {
	return "http://localhost:8080/redirecturi"
}

func (OsinClient) GetUserData() interface{} {
	return struct {
		Name string
	}{
		Name: "hoge",
	}
}
