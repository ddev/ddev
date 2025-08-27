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
	"strconv"

	"golang.org/x/sys/windows/registry"
)

func HasNativeColorSupport(stream interface{}) bool {
	return isWindows10orMore()
}

func isWindows10orMore() bool {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()

	cv, _, err := k.GetStringValue("CurrentVersion")
	if err != nil {
		return false
	}
	version, err := strconv.ParseFloat(cv, 32)
	if err != nil {
		return false
	}

	// 6.1	Windows 7 / Windows Server 2008 R2
	// 6.2	Windows 8 / Windows Server 2012
	// 6.3	Windows 8.1 / Windows Server 2012 R2
	// 10.0	Windows 10
	// But some Windows 10 systems (at least mine) return 6.3 ...
	return version >= 6.3
}
