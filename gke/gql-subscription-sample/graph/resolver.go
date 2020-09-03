package graph

import "github.com/docup/docup-api/gke/gql-subscription-sample/graph/model"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	SubscribeMessage <-chan *model.User
}
