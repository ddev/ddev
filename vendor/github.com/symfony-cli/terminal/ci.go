/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package terminal

import "os"

// from https://github.com/watson/ci-info/blob/master/vendors.json
var ciEnvs = []string{
	"APPVEYOR",
	"SYSTEM_TEAMFOUNDATIONCOLLECTIONURI",
	"bamboo_planKey",
	"BITBUCKET_COMMIT",
	"BITRISE_IO",
	"BUDDY_WORKSPACE_ID",
	"BUILDKITE",
	"CIRCLECI",
	"CIRRUS_CI",
	"CODEBUILD_BUILD_ARN",
	"CI_NAME",
	"DRONE",
	"DSARI",
	"GITLAB_CI",
	"GO_PIPELINE_LABEL",
	"HUDSON_URL",
	"JENKINS_URL",
	"BUILD_ID",
	"MAGNUM",
	"NETLIFY_BUILD_BASE",
	"NEVERCODE",
	"SAILCI",
	"SEMAPHORE",
	"SHIPPABLE",
	"TDDIUM",
	"STRIDER",
	"TASK_ID",
	"RUN_ID",
	"TEAMCITY_VERSION",
	"TRAVIS",
	"GITHUB_ACTIONS",
	"NOW_BUILDER",
	"APPCENTER_BUILD_ID",
}

func IsCI() bool {
	if os.Getenv("DEBIAN_FRONTEND") == "noninteractive" {
		return true
	}

	for _, env := range ciEnvs {
		if _, hasEnv := os.LookupEnv(env); hasEnv {
			return true
		}
	}

	return false
}
