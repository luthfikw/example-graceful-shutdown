package component

import (
	"time"

	"github.com/koinworks/asgard-heimdal/libs/logger"
)

type DisposableComponent interface {
	Dispose() error
}

type Component struct {
	Label           string
	DisposeDuration time.Duration
	DisposeError    error
}

func (ox *Component) Dispose() error {
	logger.Infof("dispossing '%s'...", ox.Label)

	time.Sleep(ox.DisposeDuration)
	if ox.DisposeError != nil {
		return ox.DisposeError
	}

	logger.Infof("disposing of '%s' has been completed.", ox.Label)
	return nil
}
