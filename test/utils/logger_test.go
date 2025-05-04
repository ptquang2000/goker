package test

import (
	utils "goker/internal/utils"
	"testing"
)

func TestLogger(t *testing.T) {
	utils.InitLogger()
	utils.LogDebug("hello world")
	utils.LogInfo("hello world")
	utils.LogWarn("hello world")
	utils.LogError("hello world")
}
