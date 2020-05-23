package env

import (
	"sync"

	"github.com/kelseyhightower/envconfig"
)

type Env struct {
	HTTPServerPort string `envconfig:"HTTP_SERVER_PORT" default:""`
	AllowedOrigins string `envconfig:"ALLOWED_ORIGINS" default:""`
}

var (
	env  Env
	err  error
	once sync.Once
)

// Process returns Env.
func Process() (Env, error) {
	once.Do(func() {
		err = envconfig.Process("", &env)
	})
	return env, err
}
