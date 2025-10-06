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

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var (
	Logger    zerolog.Logger
	LogLevels = map[int]zerolog.Level{
		1: zerolog.ErrorLevel,
		2: zerolog.WarnLevel,
		3: zerolog.InfoLevel,
		4: zerolog.DebugLevel,
		5: zerolog.TraceLevel,
	}
)

func SetLogLevel(level int) error {
	if severity, ok := LogLevels[level]; ok {
		Logger = Logger.Level(severity)
		return nil
	}
	return errors.Errorf("The provided verbosity level '%d' is not in the range [1,4]", level)
}

func IsVerbose() bool {
	return Logger.GetLevel() < zerolog.ErrorLevel
}

func IsDebug() bool {
	return Logger.GetLevel() == zerolog.TraceLevel
}

func GetLogLevel() int {
	severity := Logger.GetLevel()
	for level, sev := range LogLevels {
		if sev == severity {
			return level
		}
	}
	return 1
}

func init() {
	Logger = zerolog.New(zerolog.ConsoleWriter{Out: Stderr}).Level(zerolog.ErrorLevel).With().Timestamp().Logger()
}
