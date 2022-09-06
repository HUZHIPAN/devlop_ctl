package action

import (
	"lwapp/src/common"
	"testing"
)

func TestRunContainer(t *testing.T) {
	common.SetLwopsVolume("../../lwops")
	// RunContainer()
}

func TestRunContainerCommand(t *testing.T) {
	RunContainerCommand("php -v", 1)
}

func TestRunContainerShellScript(t *testing.T) {
	RunContainerShellScript("/itops/init.sh")
}
