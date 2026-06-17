package system

import (
	"fmt"
	"os"
	"path/filepath"
)

func MustHaveEnv(envVar string) string {
	val := os.Getenv(envVar)

	if val == "" {
		panic(fmt.Sprintf("Need '%s' for the test", envVar))
	}

	return val
}

func BackupDirWithTimestamp(deploymentName string) string {
	return fmt.Sprintf("%s_*T*Z", deploymentName)
}

func DeploymentWithName(name string) Deployment {
	testEnv := MustHaveEnv("TEST_ENV")
	fixturesDir := MustHaveEnv("FIXTURES_DIR")

	return NewDeployment(fmt.Sprintf("%s-%s", name, testEnv), filepath.Join(fixturesDir, fmt.Sprintf("%s.yml", name)))
}

func WriteEnvVarToTempFile(key string) (string, error) {
	contents := MustHaveEnv(key)

	file, err := os.CreateTemp("", "bbr-system-test")
	if err != nil {
		return "", err
	}
	defer file.Close() //nolint:errcheck

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
