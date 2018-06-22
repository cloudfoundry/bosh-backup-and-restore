package system

import (
	"fmt"
	"os"
)

var RedisDeployment = DeploymentWithName("redis")
var RedisWindowsDeployment = DeploymentWithName("redis-windows")
var RedisWithMetadataDeployment = DeploymentWithName("redis-with-metadata")
var RedisWithMissingScriptDeployment = DeploymentWithName("redis-with-missing-script")
var JumpboxDeployment = DeploymentWithName("jumpbox")
var JumpboxWindowsDeployment = DeploymentWithName("jumpbox-windows")
var JumpboxInstance = JumpboxDeployment.Instance("jumpbox", "0")
var JumpboxWindowsInstance = JumpboxDeployment.Instance("jumpbox", "0")
var RedisSlowBackupDeployment = DeploymentWithName("redis-with-slow-backup")
var RedisWithLockingOrderDeployment = DeploymentWithName("redis-with-locking-order")
var ManyBbrJobsDeployment = DeploymentWithName("many-bbr-jobs")

func MustHaveEnv(keyname string) string {
	val := os.Getenv(keyname)

	if val == "" {
		panic("Need " + keyname + " for the test")
	}

	return val
}

func BackupDirWithTimestamp(deploymentName string) string {
	return fmt.Sprintf("%s_*T*Z", deploymentName)
}

func DeploymentWithName(name string) Deployment {
	return NewDeployment(name+"-"+MustHaveEnv("TEST_ENV"), "../../fixtures/"+name+".yml")
}
