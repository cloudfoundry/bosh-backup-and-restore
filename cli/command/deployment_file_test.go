package command

import "testing"

func TestLoggerRejectsInvalidDeploymentNames(t *testing.T) {
	for _, name := range []string{"../outside", "/absolute", "..", ".", `..\outside`, ""} {
		if _, _, _, err := createLogger("now", t.TempDir(), name, false); err == nil {
			t.Fatalf("accepted deployment %q", name)
		}
	}
	if _, _, _, err := createLogger("now", t.TempDir(), "normal-deployment.1", false); err != nil {
		t.Fatal(err)
	}
}
