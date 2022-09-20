package util

import "testing"

func TestChownAll(t *testing.T) {

	ChownAll("../../demo", 1000, 1000)
}

func TestGetLocalMac(t *testing.T) {
	GetLocalMac()
}

func TestCheckPid(t *testing.T) {
	CheckPid(15160)
}

func TestRunCommandWithCli(t *testing.T) {

	// RunCommandWithCli("docker", "exec", "-it", "lwops_web_itops", "bash")
	RunCommandWithCli("node")

}

func TestCopyDirectoryAll(t *testing.T) {
	CopyDirectoryAll("../../demo/kkk1", "../../demo/kkk2", []string{"56"}, []string{".php"})
}
