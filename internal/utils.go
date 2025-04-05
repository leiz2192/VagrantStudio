package internal

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"fyne.io/fyne/v2/widget"
)

func IsDirectoryEmpty(dir string) (bool, error) {
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return false, err
	}

	d, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer d.Close()

	_, err = d.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

func IsExistedFile(path string) bool {
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func IsExistedDir(path string) bool {
	if path == "" {
		return false
	}
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func RunCommand(cmd *exec.Cmd, output *widget.Entry) error {
	if cmd == nil || output == nil {
		return nil
	}
	output.Append("\n")
	if cmd.Dir != "" {
		output.Append("\n$ cd " + cmd.Dir)
	}
	output.Append("\n$ " + cmd.String())
	// 获取命令的标准输出和标准错误的管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		output.Append("\nError: " + err.Error())
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		output.Append("\nError: " + err.Error())
		return err
	}

	// 启动命令
	err = cmd.Start()
	if err != nil {
		output.Append("\nError: " + err.Error())
		return err
	}

	// 创建扫描器实时读取输出
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			output.Append("\n" + scanner.Text())
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			output.Append("\n" + scanner.Text())
		}
	}()

	// 等待命令执行完成
	err = cmd.Wait()
	if err != nil {
		output.Append("\nError: " + err.Error())

	}
	return err
}

func NewCommand(s string, dir string) *exec.Cmd {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return nil
	}
	cmd := exec.Command(fields[0], fields[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if dir != "" {
		cmd.Dir = dir
	}
	return cmd
}

func EditFile(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// 在 Windows 系统上使用 "start" 命令打开文件
		cmd = exec.Command("cmd", "/c", "start", filePath)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	case "darwin":
		// 在 macOS 系统上使用 "open" 命令打开文件
		cmd = exec.Command("open", filePath)
	case "linux":
		// 在 Linux 系统上使用 "xdg-open" 命令打开文件
		cmd = exec.Command("xdg-open", filePath)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// 执行命令
	return cmd.Run()
}
