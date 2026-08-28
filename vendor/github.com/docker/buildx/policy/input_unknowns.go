package policy

import (
	"fmt"
	"strings"
)

func (inp *Input) setUnknowns(unknowns []string) {
	if inp == nil {
		return
	}
	if len(unknowns) == 0 {
		inp.unknowns = nil
		return
	}
	out := make([]string, 0, len(unknowns))
	for _, u := range unknowns {
		v := strings.TrimPrefix(u, "input.")
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	inp.unknowns = out
}

func (inp Input) Unknowns() []string {
	var refs []string
	collectInputUnknowns(inp, "input", &refs)
	return refs
}

func collectInputUnknowns(inp Input, prefix string, refs *[]string) {
	for _, u := range inp.unknowns {
		if u == "" {
			continue
		}
		*refs = append(*refs, prefix+"."+u)
	}
	if inp.Image == nil || inp.Image.Provenance == nil {
		return
	}
	for i := range inp.Image.Provenance.Materials {
		childPrefix := fmt.Sprintf("%s.image.provenance.materials[%d]", prefix, i)
		collectInputUnknowns(inp.Image.Provenance.Materials[i], childPrefix, refs)
	}
}
