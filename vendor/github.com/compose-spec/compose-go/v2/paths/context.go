/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package paths

import (
	"fmt"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
)

func (r *relativePathsResolver) absContextPath(value any) (any, error) {
	v, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type %T", value)
	}
	if strings.Contains(v, "://") { // `docker-image://` or any builder specific context type
		return v, nil
	}
	if strings.HasPrefix(v, types.ServicePrefix) { // `docker-image://` or any builder specific context type
		return v, nil
	}
	if isRemoteContext(v) {
		return v, nil
	}
	return r.absPath(v)
}

// absAdditionalContextPath resolves an additional build context. The value of a
// mapping may be a number or a boolean, which the schema accepts and the model
// reads as the text it is written with: it is resolved like that text, as it
// is when written in the list form. A null one stays null.
func (r *relativePathsResolver) absAdditionalContextPath(value any) (any, error) {
	switch value.(type) {
	case nil:
		return nil, nil
	case int, int64, uint64, float64, bool:
		value = fmt.Sprint(value)
	}
	return r.absContextPath(value)
}

// isRemoteContext returns true if the value is a Git reference or HTTP(S) URL.
//
// Any other value is assumed to be a local filesystem path and returns false.
//
// See: https://github.com/moby/buildkit/blob/18fc875d9bfd6e065cd8211abc639434ba65aa56/frontend/dockerui/context.go#L76-L79
func isRemoteContext(maybeURL string) bool {
	for _, prefix := range []string{"https://", "http://", "git://", "ssh://", "github.com/", "git@"} {
		if strings.HasPrefix(maybeURL, prefix) {
			return true
		}
	}
	return false
}
