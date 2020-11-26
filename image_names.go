package main

import "path"

type ImageNames []string

func (ims ImageNames) Primary() string {
	return ims[0]
}

func (ims ImageNames) Derive(registry string) ImageNames {
	out := make(ImageNames, len(ims), len(ims))
	for i, im := range ims {
		out[i] = path.Join(registry, im)
	}
	return out
}
