package main

import (
	"context"
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
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

const defaultPort = "8081"

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

	//// firestoreClient client
	//fsc := firestorm.New(firestoreClient, "ID", "")
	//
	//// OAuth server
	//storage := &OsinStorage{
	//	Ctx:       ctx,
	//	Firestore: firestoreClient,
	//	Fsc:       fsc,
	//}
	//cfg := osin.NewServerConfig()
	//cfg.AllowClientSecretInParams = true
	//OAuthServer := osin.NewServer(cfg, storage)

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
		r.Get("/redirecturi", func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			form := url.Values{}
			form.Add("code", r.FormValue("code"))
			form.Add("grant_type", "authorization_code")
			form.Add("redirect_uri", "http://c02c6157md6t.local:8081/redirecturi")
			form.Add("client_id", "qknio6kHxTHOqQZuWwd5")
			form.Add("client_secret", "c4f546a0a5e2390a573462bb67d411f3c0ba45377524b497934c57b8944c637e")
			request, error := http.NewRequest("POST", "http://localhost:8080/api/v1/token", strings.NewReader(form.Encode()))
			if error != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
			request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
			defer response.Body.Close()
			bin, err := ioutil.ReadAll(response.Body)
			if err != nil {
				w.WriteHeader(500)
				w.Write([]byte(err.Error()))
				return
			}
			res := string(bin)
			fmt.Printf("%s", res)
			w.WriteHeader(200)
			w.Header().Add("content-type", "text/html")
			w.Write([]byte("<html>" + res + "<br /><a href='http://localhost:3001'>go back</a></html>"))
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
	Ctx       context.Context
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
	o, err := FindClient(it.Ctx, it.Firestore, id)
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
	ID                   string
	AuthorizeDataPayload string
}

type AuthorizeDataUserData struct {
	Token *auth.Token
}

func (it *OsinStorage) SaveAuthorize(d *osin.AuthorizeData) error {
	abin, err := json.Marshal(d)
	if err != nil {
		return fmt.Errorf("failed Marshal")
	}
	a := &Authorized{
		ID:                   d.Code,
		AuthorizeDataPayload: string(abin),
	}
	f := it.Fsc.NewRequest().CreateEntities(it.Ctx, a)
	if err := f(); err != nil {
		return err
	}
	return nil
}

func (it *OsinStorage) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	a := &Authorized{
		ID: code,
	}
	_, err := it.Fsc.NewRequest().GetEntities(it.Ctx, a)()
	if err != nil {
		return nil, fmt.Errorf("failed GetEntities")
	}

	type Client struct {
		ID string
	}

	type AuthorizeDataUnmarshal struct {
		Client              Client
		Code                string
		ExpiresIn           int32
		Scope               string
		RedirectUri         string
		State               string
		CreatedAt           time.Time
		UserData            interface{}
		CodeChallenge       string
		CodeChallengeMethod string
	}

	d := &AuthorizeDataUnmarshal{}
	if err := json.Unmarshal([]byte(a.AuthorizeDataPayload), d); err != nil {
		return nil, fmt.Errorf("failed Unmarshal")
	}
	c, err := it.GetClient(d.Client.ID)
	if err != nil {
		return nil, err
	}

	ad := &osin.AuthorizeData{
		Client:              c,
		Code:                d.Code,
		ExpiresIn:           d.ExpiresIn,
		Scope:               d.Scope,
		RedirectUri:         d.RedirectUri,
		State:               d.State,
		CreatedAt:           d.CreatedAt,
		UserData:            d.UserData,
		CodeChallenge:       d.CodeChallenge,
		CodeChallengeMethod: d.CodeChallengeMethod,
	}
	return ad, nil
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
