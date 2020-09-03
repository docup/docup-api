package errutil

import (
	"fmt"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHasCode(t *testing.T) {
	type args struct {
		err  error
		code codes.Code
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "match",
			args: args{
				err:  status.Error(codes.NotFound, "not found"),
				code: codes.NotFound,
			},
			want: true,
		},
		{
			name: "match",
			args: args{
				err:  status.Error(codes.AlreadyExists, "not found"),
				code: codes.AlreadyExists,
			},
			want: true,
		},
		{
			name: "not match",
			args: args{
				err:  status.Error(codes.NotFound, "not found"),
				code: codes.AlreadyExists,
			},
			want: false,
		},
		{
			name: "not match",
			args: args{
				err:  fmt.Errorf("not found"),
				code: codes.AlreadyExists,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasCode(tt.args.err, tt.args.code); got != tt.want {
				t.Errorf("HasCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "true", args: args{err: status.Error(codes.NotFound, "not found")}, want: true},
		{name: "false", args: args{err: status.Error(codes.AlreadyExists, "not found")}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.args.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
