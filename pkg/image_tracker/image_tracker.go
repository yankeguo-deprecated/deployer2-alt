package image_tracker

import (
	"github.com/acicn/deployer2/pkg/cmds"
	"log"
	"sync"
)

type ImageTracker interface {
	Add(name string)
	DeleteAll()
}

type imageTracker struct {
	l      sync.Locker
	images map[string]struct{}
}

func (i *imageTracker) Add(name string) {
	i.l.Lock()
	defer i.l.Unlock()
	i.images[name] = struct{}{}
}

func (i *imageTracker) DeleteAll() {
	log.Println("清理镜像")
	for name := range i.images {
		_ = cmds.DockerRemoveImage(name)
	}
}

func New() ImageTracker {
	return &imageTracker{
		l:      &sync.Mutex{},
		images: map[string]struct{}{},
	}
}
