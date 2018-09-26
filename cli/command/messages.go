package command

const backupSigintQuestion = "Stopping a backup can leave the system in bad state. Are you sure you want to cancel? [yes/no]"
const backupStdinErrorMessage = "Couldn't read from Stdin, if you still want to stop the backup send SIGTERM."
const backupCleanupAdvisedNotice = "It is recommended that you run `bbr backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."
const backupCleanupAllDeploymentsAdvisedNotice = "It is recommended that you run `bbr deployment --all-deployments backup-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."

const restoreSigintQuestion = "Stopping a restore can leave the system in bad state. Are you sure you want to cancel? [yes/no]"
const restoreStdinErrorMessage = "Couldn't read from Stdin, if you still want to stop the restore send SIGTERM."
const restoreCleanupAdvisedNotice = "It is recommended that you run `bbr restore-cleanup` to ensure that any temp files are cleaned up and all jobs are unlocked."
