package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"net/http"
	"os"
	"strings"
	"time"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2beta3"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/auth"
	"github.com/docup/docup-api/env"
	"github.com/docup/docup-api/log"
	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jws"
	taskspb "google.golang.org/genproto/googleapis/cloud/tasks/v2beta3"
)

func main() {
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	ctx := context.Background()

	// Env
	env, err := env.Process()
	if err != nil {
		stdlog.Fatalf(err.Error())
	}

	// Logger
	logger := log.NewStandardLogger()
	log.SetDefaultLogger(logger)
	defer logger.Close()

	// sample for cloud tasks. create http task
	if false {
		_, err := createHTTPTaskWithToken(
			"docup-269111",
			"asia-northeast1",
			"default",
			//"https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app/cloudtask",
			"https://19c8d168.ngrok.io/cloudtasks",
			"docup-269111@appspot.gserviceaccount.com",
			"hoge")
		if err != nil {
			panic(err)
		}
	}

	// sample for jwt generation from service account secret key
	if false {
		jwt, err := generateJWT("soilworks-expt-01-266813-3271e161bc6e.json",
			"docup-269111@appspot.gserviceaccount.com",
			"https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app",
			3600)
		if err != nil {
			panic(err)
		}
		fmt.Println(jwt)

		res, err := makeJWTRequest(jwt, "https://endpoints-runtime-serverless-g7vn673b6q-an.a.run.app/cloudtasks")
		if err != nil {
			panic(err)
		}
		fmt.Println(res)
		return
	}

	tokenVerifier, err := NewFirebaseAuthTokenVerifier(ctx, "docup-269111")
	if err != nil {
		stdlog.Fatalf(err.Error())
	}

	r := chi.NewRouter()

	// Basic CORS
	// for more ideas, see: https://developer.github.com/v3/#cross-origin-resource-sharing
	if env.AllowedOrigins != "" {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: strings.Split(env.AllowedOrigins, ","),
			// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))
	}

	// routing
	{
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

			r.Get("/api/private", func(w http.ResponseWriter, r *http.Request) {
				idToken := r.Context().Value("GoogleIDToken")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"message": "now you see private. token:` + fmt.Sprintf("%+v", idToken) + `"}`))
				if err != nil {
					http.Error(w, http.StatusText(500), 500)
					return
				}
			})
		})

		// public routes
		r.Group(func(r chi.Router) {
			r.Get("/api/guest", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, err := w.Write([]byte(`{"message": "now you see as guest"}`))
				if err != nil {
					http.Error(w, http.StatusText(500), 500)
					return
				}
			})
		})

		////
		//r.Get("/api", func(w http.ResponseWriter, r *http.Request) {
		//	//w.Header().Set("Access-Control-Allow-Origin", "*")
		//	for key, vals := range r.Header {
		//		for _, val := range vals {
		//			w.Write([]byte(key + ":" + val + "<br />"))
		//		}
		//	}
		//	w.Write([]byte("hello api"))
		//})
		////
		//r.Get("/guest", func(w http.ResponseWriter, r *http.Request) {
		//	for key, vals := range r.Header {
		//		for _, val := range vals {
		//			w.Write([]byte(key + ":" + val + "<br />"))
		//		}
		//	}
		//	w.Write([]byte("hello root"))
		//})
		//
		r.Post("/api/cloudtasks", func(w http.ResponseWriter, r *http.Request) {
			stdlog.Println("start cloudtasks")
			for key, vals := range r.Header {
				for _, val := range vals {
					if key == "Authorization" {
						w.Write([]byte(key + ":" + val + "<br />"))
					}
					stdlog.Println(key + ":" + val)
				}
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("hello cloudtasks"))
		})
	}

	port := env.HTTPServerPort
	if port == "" {
		port = os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
	}

	host := env.HTTPServerHost
	if err := http.ListenAndServe(host+":"+port, r); err != nil {
		stdlog.Fatal(err)
	}
}

func generateJWT(saKeyfile, saEmail, audience string, expiryLength int64) (string, error) {
	now := time.Now().Unix()

	// Build the JWT payload.
	jwt := &jws.ClaimSet{
		Iat: now,
		// expires after 'expiryLength' seconds.
		Exp: now + expiryLength,
		// Iss must match 'issuer' in the security configuration in your
		// swagger spec (e.g. service account email). It can be any string.
		Iss: saEmail,
		// Aud must be either your Endpoints service name, or match the value
		// specified as the 'x-google-audience' in the OpenAPI document.
		Aud: audience,
		// Sub and Email should match the service account's email address.
		Sub:           saEmail,
		PrivateClaims: map[string]interface{}{"email": saEmail},
	}
	jwsHeader := &jws.Header{
		Algorithm: "RS256",
		Typ:       "JWT",
	}

	// Extract the RSA private key from the service account keyfile.
	sa, err := ioutil.ReadFile(saKeyfile)
	if err != nil {
		return "", fmt.Errorf("Could not read service account file: %v", err)
	}
	conf, err := google.JWTConfigFromJSON(sa)
	if err != nil {
		return "", fmt.Errorf("Could not parse service account JSON: %v", err)
	}
	block, _ := pem.Decode(conf.PrivateKey)
	parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("private key parse error: %v", err)
	}
	rsaKey, ok := parsedKey.(*rsa.PrivateKey)
	// Sign the JWT with the service account's private key.
	if !ok {
		return "", errors.New("private key failed rsa.PrivateKey type assertion")
	}
	return jws.Encode(jwsHeader, jwt, rsaKey)
}

// makeJWTRequest sends an authorized request to your deployed endpoint.
func makeJWTRequest(signedJWT, url string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Add("Authorization", "Bearer "+signedJWT)
	req.Header.Add("content-type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}
	defer response.Body.Close()
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to parse HTTP response: %v", err)
	}
	return string(responseData), nil
}

// createHTTPTaskWithToken constructs a task with a authorization token
// and HTTP target then adds it to a Queue.
func createHTTPTaskWithToken(projectID, locationID, queueID, url, email, message string) (*taskspb.Task, error) {
	// Create a new Cloud Tasks client instance.
	// See https://godoc.org/cloud.google.com/go/cloudtasks/apiv2beta3
	ctx := context.Background()
	client, err := cloudtasks.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewClient: %v", err)
	}

	// Build the Task queue path.
	queuePath := fmt.Sprintf("projects/%s/locations/%s/queues/%s", projectID, locationID, queueID)

	ah := &taskspb.HttpRequest_OidcToken{
		OidcToken: &taskspb.OidcToken{
			ServiceAccountEmail: email,
		},
	}
	t := ah.OidcToken.String()
	fmt.Println(t)

	// Build the Task payload.
	// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2beta3#CreateTaskRequest
	req := &taskspb.CreateTaskRequest{
		Parent: queuePath,
		Task: &taskspb.Task{
			// https://godoc.org/google.golang.org/genproto/googleapis/cloud/tasks/v2beta3#HttpRequest
			PayloadType: &taskspb.Task_HttpRequest{
				HttpRequest: &taskspb.HttpRequest{
					HttpMethod: taskspb.HttpMethod_POST,
					Url:        url,
				},
			},
		},
	}

	// Add a payload message if one is present.
	req.Task.GetHttpRequest().Body = []byte(message)

	createdTask, err := client.CreateTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cloudtasks.CreateTask: %v", err)
	}

	return createdTask, nil
}

type FirebaseAuthTokenVerifier struct {
	client *auth.Client
}

func NewFirebaseAuthTokenVerifier(ctx context.Context, projectID string) (*FirebaseAuthTokenVerifier, error) {
	config := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, config)
	if err != nil {
		return nil, err
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	return &FirebaseAuthTokenVerifier{
		client: client,
	}, nil
}

func (it *FirebaseAuthTokenVerifier) Verify(ctx context.Context, r *http.Request) (*auth.Token, error) {
	reqToken := r.Header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")
	if len(splitToken) != 2 {
		return nil, fmt.Errorf("Authorization bearer required")
	}
	idToken := strings.TrimSpace(splitToken[1])
	return it.VerifyToken(ctx, idToken)
}

func (it *FirebaseAuthTokenVerifier) VerifyToken(ctx context.Context, jwtToken string) (*auth.Token, error) {
	return it.client.VerifyIDToken(ctx, jwtToken)
}
