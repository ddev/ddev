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
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Color struct {
	foreground string
	background string
	options    []string
}

var (
	defaultANSIColors = map[string]string{
		"black":   "0",
		"red":     "1",
		"green":   "2",
		"yellow":  "3",
		"blue":    "4",
		"magenta": "5",
		"cyan":    "6",
		"white":   "7",
		"default": "9",
	}
	availableANSIOptions = map[string]string{
		"bold":       "1",
		"underscore": "4",
		"blink":      "5",
		"reverse":    "7",
		"conceal":    "8",
	}
	unsetANSIOptions = map[string]string{
		"1": "22",
		"4": "24",
		"5": "25",
		"7": "27",
		"8": "28",
	}
)

func NewColor(foreground, background string, options []string) (*Color, error) {
	foreground, err := parseColor(foreground)
	if err != nil {
		return nil, err
	}
	background, err = parseColor(background)
	if err != nil {
		return nil, err
	}
	color := &Color{
		foreground: foreground,
		background: background,
		options:    options,
	}
	color.options = []string{}
	seen := map[string]bool{}
	for _, option := range options {
		if code, ok := availableANSIOptions[option]; !ok {
			return nil, errors.Errorf("invalid option specified: %s.", option)
		} else if _, ok := seen[code]; !ok {
			color.options = append(color.options, code)
			seen[code] = true
		}
	}
	sort.Strings(color.options)
	return color, nil
}

func (c *Color) Apply(text []byte) []byte {
	return append(c.Set(), append(text, c.Unset()...)...)
}

func (c *Color) Set() []byte {
	setCodes := []string{}
	if c.foreground != "" {
		setCodes = append(setCodes, "3"+c.foreground)
	}
	if c.background != "" {
		setCodes = append(setCodes, "4"+c.background)
	}
	setCodes = append(setCodes, c.options...)
	if len(setCodes) == 0 {
		return nil
	}

	buf := bytes.NewBuffer([]byte{})
	buf.WriteString("\033[")
	buf.WriteString(strings.Join(setCodes, ";"))
	buf.WriteByte('m')
	return buf.Bytes()
}

func (c *Color) Unset() []byte {
	unsetCodes := []string{}
	if c.foreground != "" {
		unsetCodes = append(unsetCodes, "39")
	}
	if c.background != "" {
		unsetCodes = append(unsetCodes, "49")
	}
	for _, option := range c.options {
		unsetCodes = append(unsetCodes, unsetANSIOptions[option])
	}
	if len(unsetCodes) == 0 {
		return nil
	}

	buf := bytes.NewBuffer([]byte{})
	buf.WriteString("\033[")
	buf.WriteString(strings.Join(unsetCodes, ";"))
	buf.WriteByte('m')
	return buf.Bytes()
}

func parseColor(color string) (string, error) {
	if color == "" {
		return "", nil
	}

	if color[0] == '#' {
		var r, g, b int
		if len(color) == 4 {
			_, err := fmt.Sscanf(color, "#%1x%1x%1x", &r, &g, &b)
			if err != nil {
				return "", errors.Errorf(fmt.Sprintf("invalid \"%s\" color", color))
			}
			r *= 17
			g *= 17
			b *= 17
		} else if len(color) == 7 {
			_, err := fmt.Sscanf(color, "#%02x%02x%02x", &r, &g, &b)
			if err != nil {
				return "", errors.Errorf(fmt.Sprintf("invalid \"%s\" color", color))
			}
		} else {
			return "", errors.Errorf(fmt.Sprintf("invalid \"%s\" color", color))
		}

		return convertHexColorToAnsi(r, g, b), nil
	}

	c, ok := defaultANSIColors[color]
	if !ok {
		return "", errors.Errorf(fmt.Sprintf("invalid \"%s\" color", color))
	}

	return c, nil
}

func convertHexColorToAnsi(r, g, b int) string {
	// see https://github.com/termstandard/colors/ for more information about true color support
	if os.Getenv("COLORTERM") != "truecolor" {
		return degradeHexColorToAnsi(r, g, b)
	}

	return fmt.Sprintf("8;2;%d;%d;%d", r, g, b)
}

func degradeHexColorToAnsi(r, g, b int) string {
	if getColorSaturation(r, g, b)/50 == 0 {
		return "0"
	}

	return strconv.Itoa((b / 255 << 2) | (g / 255 << 1) | r/255)
}

func getColorSaturation(r, g, b int) int {
	r = r / 255
	g = g / 255
	b = b / 255
	max := r
	if g > max {
		max = g
	}
	if b > max {
		max = b
	}
	min := r
	if g < min {
		min = g
	}
	if b < min {
		min = b
	}
	diff := max - min
	if diff == 0 {
		return 0
	}
	return diff * 100 / max
}
