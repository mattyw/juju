// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charmcmd

import (
	"github.com/juju/cmd"
)

var charmDoc = `
"juju charm" is the the juju CLI equivalent of the "charm" command used
by charm authors, though only applicable functionality is mirrored.
`

const charmPurpose = "interact with charms"

// NewSuperCommand returns a new charm super-command.
func NewSuperCommand() *cmd.SuperCommand {
	charmCmd := cmd.NewSuperCommand(
		cmd.SuperCommandParams{
			Name:        "charm",
			Doc:         charmDoc,
			UsagePrefix: "juju",
			Purpose:     charmPurpose,
		},
	)

	// Sub-commands may be registered directly here, like so:
	//charmCmd.Register(newXXXCommand(spec))

	// ...or externally via RegisterSubCommand().
	/*
		for _, newSubCommand := range registeredSubCommands {
			command := newSubCommand()
			charmCmd.Register(command)
		}
	*/

	return charmCmd
}
