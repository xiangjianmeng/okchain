// Copyright The go-okchain Authors 2019,  All rights reserved.

package node

import (
	"fmt"

	"github.com/spf13/cobra"
)

func GetLicenseCmd() *cobra.Command {
	// Set the flags on the node start command.
	return licenseCmd
}

var licenseCmd = &cobra.Command{
	Use:   "license",
	Short: "Display license information",
	Long:  `Display license information`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return printLicense(nil)
	},
}

func printLicense(postrun func() error) error {
	fmt.Println(`Okc is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Okc is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with geth. If not, see <http://www.gnu.org/licenses/>.`)
	return nil
}
