package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

const (
	envDataFile = "data.json"
)

type Machine struct {
	Name string
	Path string
	Stat string
}

type Environment struct {
	list *widget.List

	mux  sync.RWMutex
	envs map[int]*Machine
}

func NewEnvironment() (*Environment, error) {
	e := &Environment{envs: map[int]*Machine{}}
	if err := e.LoadEnvsWithoutLock(envDataFile); err != nil {
		log.Println("Error loading data.json file:", err)
		return nil, err
	}

	go e.SaveEnvs()
	return e, nil
}

func (e *Environment) NewContent() *fyne.Container {
	output := widget.NewMultiLineEntry()
	output.SetText("Hello...")
	output.Disable()
	output.OnChanged = func(s string) { output.CursorRow++ }

	e.list = widget.NewList(
		func() int {
			return len(e.envs)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(
					widget.NewButton("Up", nil),
					widget.NewButton("Halt", nil),
					widget.NewButton("SSH", nil),
					widget.NewButton("...", nil),
				),
				container.NewGridWithColumns(3, widget.NewLabel(""), widget.NewLabel(""), widget.NewLabel("")),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			box := o.(*fyne.Container).Objects[0].(*fyne.Container)
			e.mux.RLock()
			machine := e.envs[i]
			e.mux.RUnlock()

			box.Objects[0].(*widget.Label).SetText(machine.Name)
			box.Objects[1].(*widget.Label).SetText(machine.Path)
			box.Objects[2].(*widget.Label).SetText(machine.Stat)

			w := fyne.CurrentApp().Driver().AllWindows()[0]
			btns := o.(*fyne.Container).Objects[1].(*fyne.Container)
			btns.Objects[0].(*widget.Button).OnTapped = func() {
				cmd := NewCommand("vagrant up", machine.Path)
				if err := RunCommand(cmd, output); err == nil {
					dialog.ShowInformation("Up", "vagrant up finished", w)
				} else {
					dialog.ShowError(err, w)
				}
				go e.RefreshMachineStat(i, machine.Path)
			}
			btns.Objects[1].(*widget.Button).OnTapped = func() {
				cmd := NewCommand("vagrant halt", machine.Path)
				if err := RunCommand(cmd, output); err == nil {
					dialog.ShowInformation("Halt", "vagrant halt finished", w)
				} else {
					dialog.ShowError(err, w)
				}
				go e.RefreshMachineStat(i, machine.Path)
			}
			btns.Objects[2].(*widget.Button).OnTapped = func() {
				switch runtime.GOOS {
				case "windows":
					cmd := exec.Command("cmd", "/C", "start", "cmd", "/K", "vagrant ssh")
					cmd.Dir = machine.Path
					if err := cmd.Start(); err != nil {
						log.Printf("Error opening SSH: %v", err)
					}
				default:
					cmd := exec.Command("xterm", "-e", "vagrant ssh")
					cmd.Dir = machine.Path
					if err := cmd.Start(); err != nil {
						cmd = exec.Command("gnome-terminal", "--", "vagrant ssh")
						cmd.Dir = machine.Path
						if err = cmd.Start(); err != nil {
							log.Printf("Failed to run vagrant ssh: %v", err)
						}
					}
				}
			}
			btns.Objects[3].(*widget.Button).OnTapped = func() {
				menu := fyne.NewMenu("Options",
					fyne.NewMenuItem("Port", func() {
						cmd := NewCommand("vagrant port", machine.Path)
						if err := RunCommand(cmd, output); err == nil {
							dialog.ShowInformation("Port", "vagrant port finished", w)
						} else {
							dialog.ShowError(err, w)
						}
					}),
					fyne.NewMenuItem("Edit", func() {
						fpath := filepath.Join(machine.Path, "Vagrantfile")
						if err := EditFile(fpath); err != nil {
							log.Printf("Error opening Vagrantfile %s: %v", fpath, err)
						}
					}),
					fyne.NewMenuItem("Reload", func() {
						cmd := NewCommand("vagrant reload", machine.Path)
						if err := RunCommand(cmd, output); err == nil {
							dialog.ShowInformation("Reload", "vagrant reload finished", w)
						} else {
							dialog.ShowError(err, w)
						}
					}),
					fyne.NewMenuItem("Remove", func() {
						dialog.ShowConfirm("Remove", "Are you sure to remove this machine?", func(confirmed bool) {
							if !confirmed {
								return
							}
							e.mux.Lock()
							delete(e.envs, i)
							e.mux.Unlock()
							e.list.Refresh()
						}, fyne.CurrentApp().Driver().AllWindows()[0])

					}),
				)
				pop := widget.NewPopUpMenu(menu, w.Canvas())
				pop.ShowAtRelativePosition(fyne.NewPos(0, 0), btns.Objects[3].(*widget.Button))
			}
		},
	)

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter env path...")
	btnBox := container.NewBorder(nil, nil, nil,
		container.NewHBox(
			widget.NewButton("Open", func() {
				dialog.NewFolderOpen(func(list fyne.ListableURI, err error) {
					if err != nil {
						dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
						return
					}
					if list == nil {
						return
					}

					input.SetText(list.Path())
				}, fyne.CurrentApp().Driver().AllWindows()[0]).Show()
			}),
			widget.NewButton("Add", func() {
				if input.Text == "" || !IsExistedDir(input.Text) || !IsExistedFile(filepath.Join(input.Text, "Vagrantfile")) {
					dialog.ShowError(errors.New("path is not a valid vagrant env"), fyne.CurrentApp().Driver().AllWindows()[0])
					return
				}

				e.mux.Lock()
				for _, m := range e.envs {
					if m.Path == input.Text {
						dialog.ShowError(errors.New("path already exists"), fyne.CurrentApp().Driver().AllWindows()[0])
						e.mux.Unlock()
						return
					}
				}
				e.envs[len(e.envs)] = &Machine{Name: filepath.Base(input.Text), Path: input.Text, Stat: ""}
				e.mux.Unlock()

				e.list.Refresh()
				go e.RefreshMachineStat(len(e.envs)-1, input.Text)
				input.SetText("")
			}),
			widget.NewButton("Refresh", func() {
				var wg sync.WaitGroup
				for i := range e.envs {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						e.RefreshMachineStat(i, e.envs[i].Path)
					}(i)
				}
				wg.Wait()
				e.list.Refresh()
			}),
		),
		input,
	)

	return container.NewBorder(btnBox, nil, nil, nil, container.NewGridWithColumns(1, e.list, container.NewScroll(output)))
}

func (e *Environment) SaveEnvs() {
	t := time.NewTicker(time.Second * 5)
	for {
		<-t.C
		e.mux.RLock()
		if err := e.SaveEnvsWithoutLock(envDataFile); err != nil {
			log.Println("Error saving data.json file:", err)
		}
		e.mux.RUnlock()
	}
}

func (e *Environment) Close() error {
	e.mux.RLock()
	defer e.mux.RUnlock()
	return e.SaveEnvsWithoutLock(envDataFile)
}

func (e *Environment) SaveEnvsWithoutLock(fpath string) error {
	file, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 使用 json.NewEncoder 将 map 编码为 JSON 并写入文件
	return json.NewEncoder(file).Encode(e.envs)
}

func (e *Environment) LoadEnvsWithoutLock(fpath string) error {
	stat, err := os.Stat(fpath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		log.Println("Error checking file existence:", err)
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("file %s is a directory", fpath)
	}
	if stat.Size() == 0 {
		return nil
	}

	file, err := os.Open(fpath)
	if err != nil {
		log.Println("Error opening data.json file:", err)
		return err
	}
	defer file.Close()

	if err = json.NewDecoder(file).Decode(&e.envs); err != nil {
		log.Println("Error decoding JSON:", err)
		return err
	}
	return nil
}

func (e *Environment) MachineStat(path string) (string, error) {
	cmd := NewCommand("vagrant status", path)
	rtn, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("vagrant status in %s Error: %s\nOutput: %s", path, err.Error(), string(rtn))
	}
	var noEmptyLines []string
	for line := range strings.SplitSeq(string(rtn), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			noEmptyLines = append(noEmptyLines, trimmed)
		}
		if len(noEmptyLines) >= 2 {
			break
		}
	}
	if len(noEmptyLines) < 2 {
		return "", fmt.Errorf("vagrant status in %s Error: no valid output\nOutput: %s", path, string(rtn))
	}
	fields := strings.Fields(noEmptyLines[1])
	if len(fields) < 2 {
		return "", fmt.Errorf("vagrant status in %s Error: no valid status\nOutput: %s", path, string(rtn))
	}
	return fields[1], nil
}

func (e *Environment) SetMachineStat(item int, path string) {
	stat, err := e.MachineStat(path)
	if err != nil {
		log.Println("Error refreshing machine stat:", err)
		return
	}
	e.mux.Lock()
	e.envs[item].Stat = stat
	e.mux.Unlock()
}

func (e *Environment) RefreshMachineStat(item int, path string) {
	e.SetMachineStat(item, path)
	e.list.Refresh()
}
