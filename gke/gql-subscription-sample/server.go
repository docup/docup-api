package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/docup/docup-api/gke/gql-subscription-sample/graph"
	"github.com/docup/docup-api/gke/gql-subscription-sample/graph/generated"
	"github.com/docup/docup-api/gke/gql-subscription-sample/graph/model"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
)

const defaultPort = "8080"

func main() {
	ctx := context.Background()

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	var u = make(chan *model.User)

	// pubsubでpullしてu (chan *model.User) にメッセージを流す
	{
		client, err := pubsub.NewClient(ctx, "docup-269111")
		if err != nil {
			log.Fatalf("pubsub.NewClient: %v", err)
		}
		sub := client.Subscription("gql-subscription-sample-subcription")

		go func() {
			fmt.Printf("start subscription pull\n")
			err = sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
				fmt.Printf("received:%+v", msg)
				u <- &model.User{
					ID:   "pubsubID:" + msg.ID,
					Name: "pubsubMessage" + string(msg.Data),
				}
				msg.Ack()
			})
			if err != nil {
				log.Fatalf("Receive: %v", err)
			}
		}()
	}

	resolver := &graph.Resolver{
		SubscribeMessage: u,
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
	srv.Use(extension.Introspection{})

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", c.Handler(srv))

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
