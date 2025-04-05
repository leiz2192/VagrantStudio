package internal

import (
	"os/exec"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
)

func TestRunCommand(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	output := widget.NewMultiLineEntry()

	// 模拟一个成功执行的命令
	cmd := exec.Command("vagrant", "version")

	RunCommand(cmd, output)

	// 断言输出是否符合预期
	assert.Contains(t, output.Text, "Installed Version:")
}

func TestRunCommand_Error(t *testing.T) {
	testApp := test.NewApp()
	defer testApp.Quit()

	output := widget.NewEntry()

	// 模拟一个执行出错的命令
	cmd := exec.Command("nonexistent_command")

	RunCommand(cmd, output)

	// 断言是否输出了错误信息
	assert.Contains(t, output.Text, "Error starting command")
}
