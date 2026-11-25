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

package loader

import (
	"math"
	"strconv"
	"strings"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/sirupsen/logrus"
)

// restartPolicyName maps the name of a restart policy, as written in `restart`
// or in `deploy.restart_policy.condition`, to the vocabulary of `restart`.
// The second result is false for a name Compose does not know, which is left
// for the platform to report.
func restartPolicyName(name string) (string, bool) {
	switch name {
	case "no", "none":
		return types.RestartPolicyNo, true
	case "always", "any":
		return types.RestartPolicyAlways, true
	case types.RestartPolicyOnFailure, types.RestartPolicyUnlessStopped:
		return name, true
	}
	return "", false
}

// mergeRestart merges `deploy.restart_policy` into `restart`: the two are
// spellings of one policy, where `condition` is the name of the policy and
// `max_attempts` its retry count (`restart: on-failure:3`), while `delay` and
// `window` have no counterpart in `restart`. Each attribute may thus complete
// the other. An attribute that is not set does not say anything: a
// `restart_policy` made of a `delay` alone leaves `restart` as it is.
//
// Where both say something different, `deploy.restart_policy` wins, as the
// spec has it ("if `restart_policy` is not set, Compose considers the `restart`
// field"), and conflict is true. Two names agree when they are the same policy
// in the vocabulary of `restart`: `any` is `always`, `none` is `no`. `any` does
// not agree with `unless-stopped`: it cannot say "unless stopped", and the
// platforms which read `restart_policy` alone (Docker Compose does) already
// turn `any` into `always`. A retry count only has a meaning for `on-failure`:
// it replaces the one of `restart` there, and it is dropped when the policy
// does not retry, as `restart` has no way to say it (`always:2` is no value).
// Zero retries is no limit, as for `on-failure:0`: it does not disagree with a
// policy that has no retry count.
//
// It returns the value `restart` takes, or "" when there is no policy to carry
// (nothing is set, or a value is left for the platform to report).
func mergeRestart(restart, condition string, maxAttempts *uint64) (merged string, conflict bool) {
	var name string
	restartName, restartCount, hasRestartCount := strings.Cut(restart, ":")
	if restart != "" {
		var known bool
		if name, known = restartPolicyName(restartName); !known {
			return "", false
		}
	}
	retries, hasRetries := parseRetries(restartCount, hasRestartCount)
	if hasRestartCount && !hasRetries {
		// `on-failure:abc`: left for the platform to report
		return "", false
	}

	if condition != "" {
		conditionName, known := restartPolicyName(condition)
		if !known {
			return "", false
		}
		if name != "" && name != conditionName {
			// another policy: the retry count of `restart` went with it
			conflict, hasRetries = true, false
		}
		name = conditionName
	}

	if maxAttempts != nil && (*maxAttempts != 0 || name == types.RestartPolicyOnFailure) {
		switch {
		case name == types.RestartPolicyOnFailure:
			if hasRetries && retries != *maxAttempts {
				conflict = true
			}
			retries, hasRetries = *maxAttempts, true
		case restart != "":
			// `restart` already pins a policy that has no retry count to
			// give: there is nothing to carry the count in
			conflict = true
		case name != "":
			// a retry count with this policy is no `restart` value: the
			// platform reports it, as it does today
			return "", false
		}
	}

	if name == "" {
		// nothing but `delay`, `window` or `max_attempts`: no policy to carry
		return "", false
	}
	if hasRetries {
		name += ":" + strconv.FormatUint(retries, 10)
	}
	return name, conflict
}

// carryRestartPolicy writes the outcome of mergeRestart in the `restart` of the
// service called name, in the canonical model. It runs when the model is
// normalized, that is once the files, `include` and `extends` are merged (either
// attribute may be changed until then). Where `restart` contradicts
// `deploy.restart_policy`, the latter wins and a warning says so. A value that
// has not the shape the schema gives it is left to be reported where the model
// is validated or bound.
func carryRestartPolicy(name string, service map[string]any) {
	deploy, _ := service["deploy"].(map[string]any)
	policy, _ := deploy["restart_policy"].(map[string]any)
	if policy == nil {
		return
	}
	maxAttempts, ok := attempts(policy["max_attempts"])
	if !ok {
		// a retry count which is not one: carrying the policy without it would
		// lose it in silence, so it is left to be reported when the model is bound
		return
	}
	restart, _ := service["restart"].(string)
	condition, _ := policy["condition"].(string)
	merged, conflict := mergeRestart(restart, condition, maxAttempts)
	if merged == "" {
		return
	}
	if conflict {
		var attrs []string
		if condition != "" {
			attrs = append(attrs, "condition: "+condition)
		}
		if maxAttempts != nil {
			attrs = append(attrs, "max_attempts: "+strconv.FormatUint(*maxAttempts, 10))
		}
		logrus.Warnf("services.%s: 'restart' (%s) and 'deploy.restart_policy' (%s) disagree: 'deploy.restart_policy' takes precedence, the restart policy is %s",
			name, restart, strings.Join(attrs, ", "), merged)
	}
	service["restart"] = merged
}

// attempts reads the `max_attempts` of a model: nil when it is not set, and
// false when it is set to something which is not a retry count.
func attempts(value any) (*uint64, bool) {
	var n uint64
	switch v := value.(type) {
	case nil:
		return nil, true
	case int:
		if v < 0 {
			return nil, false
		}
		n = uint64(v)
	case int64:
		if v < 0 {
			return nil, false
		}
		n = uint64(v)
	case uint64:
		n = v
	case float64:
		if v < 0 || v != math.Trunc(v) || v >= math.Exp2(64) {
			return nil, false
		}
		n = uint64(v)
	case string:
		parsed, ok := parseRetries(v, true)
		if !ok {
			return nil, false
		}
		n = parsed
	default:
		return nil, false
	}
	return &n, true
}

// parseRetries reads a retry count. It reports false for an absent or
// malformed count, which is left for the platform to report.
func parseRetries(value string, present bool) (uint64, bool) {
	if !present {
		return 0, false
	}
	n, err := strconv.ParseUint(value, 10, 64)
	return n, err == nil
}
