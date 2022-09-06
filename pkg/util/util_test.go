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
