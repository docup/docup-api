package main

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/docup/docup-api/env"
	"github.com/docup/docup-api/errutil"
	"github.com/docup/docup-api/graph"
	"github.com/docup/docup-api/graph/generated"
	"github.com/docup/docup-api/log2"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"
	"github.com/jschoedt/go-firestorm"
	"github.com/openshift/osin"
	//"github.com/rs/cors"
)

const defaultPort = "8080"

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
		AllowedOrigins:   []string{"http://localhost:3000", "http://c02c6157md6t.local:3000"},
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

	// firestoreClient client
	fsc := firestorm.New(firestoreClient, "ID", "")

	// OAuth server
	storage := &OsinStorage{
		Firestore: firestoreClient,
		Fsc:       fsc,
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

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: strings.Split("http://c02c6157md6t.local:3000", ","),
		// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

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
		// "http://localhost:8080/authorize?redirect_uri=http%3A%2F%2Flocalhost%3A8080%2Fredirecturi&response_type=code&client_id=qknio6kHxTHOqQZuWwd5&state=1"
		r.Get("/authorize", func(w http.ResponseWriter, r *http.Request) {
			resp := OAuthServer.NewResponse()
			defer resp.Close()
			if ar := OAuthServer.HandleAuthorizeRequest(resp, r); ar != nil {
				token, err := tokenVerifier.Verify(ctx, r)
				if err != nil {
					http.Error(w, http.StatusText(401), 401)
					w.Write([]byte(err.Error()))
					return
				}

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
				ar.UserData = token
				OAuthServer.FinishAuthorizeRequest(resp, r, ar)
				osin.OutputJSON(resp, w, r)
			} else {
				bin, err := json.Marshal(resp.Output)
				if err != nil {
					logger.Error(err.Error())
				}
				if resp.IsError {
					w.WriteHeader(http.StatusInternalServerError)
				} else {
					w.WriteHeader(http.StatusBadRequest)
				}
				w.Write(bin)
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
	Fsc       *firestorm.FSClient
}

func (it *OsinStorage) Clone() osin.Storage {
	return it
}

func (OsinStorage) Close() {
	// do nothing
}

func (it *OsinStorage) GetClient(id string) (osin.Client, error) {
	o, err := FindClient(context.Background(), it.Firestore, id)
	if err != nil {
		if errutil.IsNotFound(err) {
			return nil, osin.ErrNotFound
		}
		return nil, err
	}
	o.ID = id
	return o, nil
}

type Authorized struct {
	ID               string
	AuthorizePayload string
	TokenPayload     string
}

func (it *OsinStorage) SaveAuthorize(d *osin.AuthorizeData) error {
	token, ok := d.UserData.(*auth.Token)
	if !ok {
		return fmt.Errorf("invalid UserData")
	}
	fmt.Println(token)

	abin, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed Marshal")
	}
	tbin, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed Marshal")
	}

	a := &Authorized{
		ID:               d.Code,
		AuthorizePayload: string(abin),
		TokenPayload:     string(tbin),
	}
	f := it.Fsc.NewRequest().CreateEntities(context.Background(), a)
	if err := f(); err != nil {
		return err
	}
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

func FindClient(ctx context.Context, firestore *firestore.Client, id string) (*OsinClient, error) {
	dsnap, err := firestore.Collection("clients").Doc(id).Get(ctx)
	if err != nil {
		return nil, err
	}
	o := &OsinClient{}
	if err := dsnap.DataTo(o); err != nil {
		return nil, err
	}
	return o, nil
}

type OsinClient struct {
	ID          string      `firestore:"id"`
	Secret      string      `firestore:"secret"`
	RedirectURI string      `firestore:"redirectUri"`
	UserData    interface{} `firestore:"userData"`
}

func (it *OsinClient) GetId() string {
	return it.ID
}

func (it *OsinClient) GetSecret() string {
	return it.Secret
}

func (it *OsinClient) GetRedirectUri() string {
	return it.RedirectURI
}

func (it *OsinClient) GetUserData() interface{} {
	return it.UserData
}

func createClientSecret() string {
	return secureRandomStr(32)
}

func secureRandomStr(b int) string {
	k := make([]byte, b)
	if _, err := crand.Read(k); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", k)
}

func createClientID() string {
	return RandString(20)
}

const rs2Letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = rs2Letters[rand.Intn(len(rs2Letters))]
	}
	return string(b)
}
