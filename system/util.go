package system

import (
	"fmt"
	"io/ioutil"
	"os"
)

var RedisDeployment = DeploymentWithName("redis")
var RedisWithBackupOneRestoreAll = DeploymentWithName("redis-with-backup-one-restore-all")
var RedisWithMissingScriptDeployment = DeploymentWithName("redis-with-missing-script")
var JumpboxDeployment = DeploymentWithName("jumpbox")
var JumpboxInstance = JumpboxDeployment.Instance("jumpbox", "0")
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

func WriteEnvVarToTempFile(key string) (string, error) {
	contents := MustHaveEnv(key)

	file, err := ioutil.TempFile("", "bbr-system-test")
	if err != nil {
		return "", err
	}

	err = file.Chmod(0644)
	if err != nil {
		return "", err
	}

	_, err = file.WriteString(contents)
	if err != nil {
		return "", err
	}

	return file.Name(), nil
}
