package worker

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"gitea.com/iwakuramarie/nyat/worker/handlers"
	"gitea.com/iwakuramarie/nyat/worker/types"
)

// Guesses the appropriate worker type based on the given source string
func NewWorker(source string, logger *log.Logger) (*types.Worker, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, err
	}
	worker := types.NewWorker(logger)
	scheme := u.Scheme
	if strings.ContainsRune(scheme, '+') {
		scheme = scheme[:strings.IndexRune(scheme, '+')]
		fmt.Println(scheme)
	}
	backend, err := handlers.GetHandlerForScheme(scheme, worker)
	if err != nil {
		return nil, err
	}
	worker.Backend = backend
	return worker, nil
}
