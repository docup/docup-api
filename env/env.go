package env

import (
	"sync"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap/zapcore"
)

type Env struct {
	HTTPServerHost string        `envconfig:"HTTP_SERVER_HOST" default:""`
	HTTPServerPort string        `envconfig:"HTTP_SERVER_PORT" default:""`
	AllowedOrigins string        `envconfig:"ALLOWED_ORIGINS" default:""`
	LogLevel       zapcore.Level `envconfig:"LOG_LEVEL" default:"INFO"`
	Env            string        `envconfig:"ENV" default:"production"`
	ProjectID      string        `envconfig:"PROJECT_ID" required:"true"`
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
