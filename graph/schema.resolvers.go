package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"

	"github.com/docup/docup-api/graph/generated"
	"github.com/docup/docup-api/graph/model"
	"google.golang.org/api/iterator"
)

func (r *mutationResolver) CreateTodo(ctx context.Context, input model.NewTodo) (*model.Todo, error) {
	_, _, err := r.Firestore.Collection("users").Add(ctx, map[string]interface{}{
		"first": "Ada" + strconv.FormatUint(rand.Uint64(), 10),
		"last":  "Lovelace",
		"born":  1815,
	})
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (r *queryResolver) Todos(ctx context.Context) ([]*model.Todo, error) {
	todos := []*model.Todo{}
	iter := r.Firestore.Collection("users").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		a := struct {
			First string `json:"first"`
		}{}

		dd := doc.Data()
		MapToStruct(dd, &a)

		todos = append(todos, &model.Todo{
			Text: fmt.Sprintf("%+v", a.First),
		})
	}
	return todos, nil
}

func MapToStruct(mapVal map[string]interface{}, val interface{}) error {
	//structVal := reflect.Indirect(reflect.ValueOf(val))
	//for name, elem := range mapVal {
	//	v := structVal.FieldByName(name)
	//
	//	if v.IsValid() {
	//		v.Set(reflect.ValueOf(elem))
	//	}
	//}
	//return

	bin, err := json.Marshal(mapVal)
	if err != nil {
		return err
	}
	return json.Unmarshal(bin, val)
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
