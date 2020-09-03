package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"math/rand"
	"strconv"

	"github.com/docup/docup-api/gke/gql-subscription-sample/graph/generated"
	"github.com/docup/docup-api/gke/gql-subscription-sample/graph/model"
)

var todos = []*model.Todo{}

func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	m := &model.Todo{
		ID:   strconv.FormatUint(rand.Uint64(), 10),
		Text: input.Text,
		Done: false,
		User: &model.User{
			ID:   input.UserID,
			Name: "uname",
		},
	}
	todos = append(todos, m)
	return m, nil
}

func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	return todos, nil
}

func (r *subscriptionResolver) MessageAdded(ctx context.Context, roomName string) (<-chan *model.User, error) {
	return r.Resolver.SubscribeMessage, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation.
func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
