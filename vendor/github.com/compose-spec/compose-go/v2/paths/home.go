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
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// ExpandUser replaces a leading "~" by the home directory of the current user.
// Only "~" alone, or followed by a path separator, is the home directory: a
// "~" followed by a name ("~user/x") is the home directory of another user,
// which is not looked up, and the path is left untouched.
func ExpandUser(p string) string {
	if isHomePath(p) {
		home, err := os.UserHomeDir()
		if err != nil {
			logrus.Warn("cannot expand '~', because the environment lacks HOME")
			return p
		}
		return filepath.Join(home, p[1:])
	}
	return p
}

// isHomePath reports whether p is "~", or starts with "~" and a path
// separator.
func isHomePath(p string) bool {
	if !strings.HasPrefix(p, "~") {
		return false
	}
	if len(p) == 1 {
		return true
	}
	return p[1] == '/' || p[1] == filepath.Separator
}
