package main

const helpTextTemplate = `NAME:
   {{.Name}}{{if .Usage}} - {{.Usage}}{{end}}

USAGE:
   bbr command [arguments...] [subcommand]{{if .Version}}{{if not .HideVersion}}

VERSION:
   {{.Version}}{{end}}{{end}}{{if .Description}}

DESCRIPTION:
   {{.Description}}{{end}}{{if len .Authors}}

AUTHOR{{with $length := len .Authors}}{{if ne 1 $length}}S{{end}}{{end}}:
   {{range $index, $author := .Authors}}{{if $index}}
   {{end}}{{$author}}{{end}}{{end}}{{if .VisibleCommands}}

COMMANDS:{{range .VisibleCategories}}{{if .Name}}
   {{.Name}}:{{end}}{{range .VisibleCommands}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}{{end}}

SUBCOMMANDS:
   backup
   backup-cleanup
   restore
   restore-cleanup
   pre-backup-check{{if .Copyright}}

COPYRIGHT:
   {{.Copyright}}{{end}}
`
const backupSigintQuestion = "Stopping a backup can leave the system in bad state. Are you sure you want to cancel? [yes/no]"
const backupStdinErrorMessage = "Couldn't read from Stdin, if you still want to stop the backup send SIGTERM."
const backupCleanupAdvisedNotice = "It is recommended that you run `bbr backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."

const restoreSigintQuestion = "Stopping a restore can leave the system in bad state. Are you sure you want to cancel? [yes/no]"
const restoreStdinErrorMessage = "Couldn't read from Stdin, if you still want to stop the restore send SIGTERM."
const restoreCleanupAdvisedNotice = "It is recommended that you run `bbr restore-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."
