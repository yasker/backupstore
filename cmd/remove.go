package cmd

import (
	"github.com/urfave/cli"

	"github.com/yasker/backupstore"
	"github.com/yasker/backupstore/util"
)

var (
	BackupRemoveCmd = cli.Command{
		Name:   "rm",
		Usage:  "remove a backup in objectstore: rm <backup>",
		Action: cmdBackupRemove,
	}
)

func cmdBackupRemove(c *cli.Context) {
	if err := doBackupRemove(c); err != nil {
		panic(err)
	}
}

func doBackupRemove(c *cli.Context) error {
	if c.NArg() == 0 {
		return RequiredMissingError("backup URL")
	}
	backupURL := c.Args()[0]
	if backupURL == "" {
		return RequiredMissingError("backup URL")
	}
	backupURL = util.UnescapeURL(backupURL)

	if err := backupstore.DeleteDeltaBlockBackup(backupURL); err != nil {
		return err
	}
	return nil
}
