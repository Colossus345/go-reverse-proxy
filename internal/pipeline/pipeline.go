package pipeline

import (
	"io"
	"os"

	"github.com/Colossus345/Go-interpreter/export"
)

type Pipeline struct {
	Middlewares []export.Program
}

func New(files []string) (*Pipeline, error) {
	var result Pipeline
	for _, f := range files {
		file, err := os.Open(f)
		if err != nil {
			return nil, err
		}
		s, err := io.ReadAll(file)
		if err != nil {
			return nil, err
		}
		m, err := export.Compile(string(s))
		if err != nil {
			return nil, err
		}
		result.Middlewares = append(result.Middlewares, m)

	}
	return &result, nil
}

func (p *Pipeline) Exec(msg string, opts string) (string, error) {
	var result string = msg
	for _, m := range p.Middlewares {
		obj, err := export.Exec(m, result)
		if err != nil {
			return "", err
		}
        result = obj
	}

	return result, nil
}
