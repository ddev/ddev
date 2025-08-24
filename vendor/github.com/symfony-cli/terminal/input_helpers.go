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
	"bufio"
	"fmt"
	"strings"
)

func AskString(message string, validator func(string) (string, bool)) string {
	return AskStringDefault(message, "", validator)
}

func AskStringDefault(message, def string, validator func(string) (string, bool)) string {
	if !Stdin.IsInteractive() {
		return def
	}
	if def != "" {
		message = fmt.Sprintf("%s <question>[%s]</> ", message, def)
	}

	reader := bufio.NewReader(Stdin)
	for {
		Print(message)
		answer, readError := reader.ReadString('\n')
		if readError != nil {
			continue
		}
		answer = strings.TrimRight(answer, "\r\n")
		answer = strings.Trim(answer, " ")
		if answer == "" {
			answer = def
		}
		if answer, isValid := validator(answer); !isValid {
			continue
		} else {
			return answer
		}
	}
}

func AskConfirmation(message string, def bool) bool {
	if !Stdin.IsInteractive() {
		return def
	}
	defaultHelp, defaultAnswer := "Y/n", "yes"
	if !def {
		defaultHelp, defaultAnswer = "y/N", "no"
	}
	message = fmt.Sprintf("%s <question>[%s]</> ", message, defaultHelp)

	answer := AskString(message, func(answer string) (string, bool) {
		answer = strings.ToLower(answer)
		if answer == "" {
			return defaultAnswer, true
		}
		if answer == "y" || answer == "yes" {
			return "yes", true
		}
		if answer == "n" || answer == "no" {
			return "no", true
		}

		return answer, false
	})

	return answer == "yes"
}
