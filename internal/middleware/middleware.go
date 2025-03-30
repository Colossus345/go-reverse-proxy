package middleware

import (
	"io"
	"os"

	"github.com/Colossus345/Go-interpreter/interpreter"
)

type MiddlewareFunc func([]byte) ([]byte, error)

type Middleware interface {
	Process([]byte, map[string]interface{}) ([]byte, error)
}

type Pipeline struct {
	Middlewares []interpreter.Program
	Variables   map[string]interface{}
}

type MiddlewareConfig struct {
	Name      string                 `yaml:"path"`
	Variables map[string]interface{} `yaml:"variables"`
}

func New(configs []MiddlewareConfig) (Middleware, error) {
	var result Pipeline
	for _, config := range configs {
		result.Variables = config.Variables

		file, err := os.Open(config.Name)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		s, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}

		program := string(s)

		m, err := interpreter.Compile(program)
		if err != nil {
			return nil, err
		}
		result.Middlewares = append(result.Middlewares, m)
	}

	return &result, nil
}

func (p *Pipeline) Process(msg []byte, args map[string]interface{}) ([]byte, error) {
	var result string = string(msg)
	if args == nil {
		args = make(map[string]interface{})
	}
	for key, val := range p.Variables {
		args["__"+key] = val
	}
	args["__msg__"] = string(msg)
	for _, m := range p.Middlewares {
		obj, err := interpreter.Exec(m, args)
		if err != nil {
			return nil, err
		}
		result = obj
	}

	return []byte(result), nil
}
