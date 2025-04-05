package internal

import (
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type box struct {
	Name     string
	Provider string
	Version  string
}

func (b *box) String() string {
	return b.Name + " (" + b.Provider + ", " + b.Version + ")"
}

type Box struct {
	list  *widget.List
	boxes map[int]*box
}

func NewBox() *Box {
	return &Box{boxes: map[int]*box{}}
}

func (b *Box) NewContent() *fyne.Container {
	output := widget.NewMultiLineEntry()
	output.SetText("Hello...")
	output.Disable()
	output.OnChanged = func(s string) { output.CursorRow++ }

	b.list = widget.NewList(
		func() int {
			return len(b.boxes)
		},
		func() fyne.CanvasObject {
			return container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(widget.NewButton("InitEnv", nil), widget.NewButton("Remove", nil)),
				container.NewGridWithColumns(2, widget.NewLabel(""), widget.NewLabel("")),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			item := b.boxes[i]
			o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[0].(*widget.Label).SetText(item.Name)
			o.(*fyne.Container).Objects[0].(*fyne.Container).Objects[1].(*widget.Label).SetText(item.Provider + ", " + item.Version)

			o.(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*widget.Button).OnTapped = func() {
				output.Append("\nvagrant init " + item.String())
				entry := widget.NewEntry()
				entry.SetPlaceHolder("Enter env path...")
				content := container.NewBorder(nil, nil, widget.NewLabel("Path"), widget.NewButton("Browse", nil), entry)
				content.Objects[2].(*widget.Button).OnTapped = func() {
					dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
						if err != nil {
							return
						}
						entry.SetText(uri.Path())
					}, fyne.CurrentApp().Driver().AllWindows()[0])
				}
				d := dialog.NewCustom("InitEnv", "", content, fyne.CurrentApp().Driver().AllWindows()[0])
				d.SetButtons([]fyne.CanvasObject{
					widget.NewButton("Cancle", func() { d.Hide() }),
					widget.NewButton("Submit", func() {
						if entry.Text == "" {
							return
						}
						empty, err := IsDirectoryEmpty(entry.Text)
						if err != nil || !empty {
							dialog.ShowInformation("Error", "Path is not a valid directory", fyne.CurrentApp().Driver().AllWindows()[0])
							return
						}

						cmd := NewCommand("vagrant init "+item.Name, entry.Text)
						if err := RunCommand(cmd, output); err == nil {
							dialog.ShowInformation("InitEnv", "InitEnv finished", fyne.CurrentApp().Driver().AllWindows()[0])
						} else {
							dialog.ShowInformation("Error", "InitEnv failed", fyne.CurrentApp().Driver().AllWindows()[0])
						}
						d.Hide()
					}),
				})
				d.SetOnClosed(func() {
					fyne.CurrentApp().Driver().AllWindows()[0].Canvas().Focus(nil) // 关闭对话框后恢复焦点
				})
				d.Resize(fyne.NewSize(400, 100))
				d.Show()
			}
			o.(*fyne.Container).Objects[1].(*fyne.Container).Objects[1].(*widget.Button).OnTapped = func() {
				cmd := NewCommand("vagrant box remove "+strings.Fields(item.Name)[0], "")
				if err := RunCommand(cmd, output); err == nil {
					b.refreshBoxes()
				}
				dialog.ShowInformation("Remove", "Remove finished", fyne.CurrentApp().Driver().AllWindows()[0])
			}
		},
	)
	go b.refreshBoxes()

	input := widget.NewEntry()
	input.SetPlaceHolder("Enter name, url, or path...")
	top := container.NewBorder(nil, nil, nil,
		container.NewHBox(
			widget.NewButton("Add", func() {
				if input.Text == "" {
					return
				}
				text := strings.Trim(strings.TrimSpace(input.Text), `"`)
				var cmd *exec.Cmd
				if strings.HasSuffix(text, ".box") {
					name := strings.TrimSuffix(filepath.Base(text), ".box")
					cmd = NewCommand("vagrant box add "+name+" "+text, "")
				} else {
					cmd = NewCommand("vagrant box add "+text, "")
				}

				if err := RunCommand(cmd, output); err == nil {
					b.refreshBoxes()
					input.SetText("")
				}
				dialog.ShowInformation("Add", "Add finished", fyne.CurrentApp().Driver().AllWindows()[0])
			}),
			widget.NewButton("Refresh", func() {
				b.refreshBoxes()
			}),
		),
		input,
	)

	return container.NewBorder(top, nil, nil, nil, container.NewGridWithColumns(1, b.list, container.NewScroll(output)))
}

func (b *Box) refreshBoxes() {
	cmd := NewCommand("vagrant box list", "")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	boxes := make(map[int]*box, 0)
	for i, one := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.FieldsFunc(one, func(r rune) bool {
			return r == '(' || r == ')' || r == ',' || r == ' '
		})
		if len(fields) < 3 {
			continue
		}
		boxes[i] = &box{Name: fields[0], Provider: fields[1], Version: fields[2]}
	}
	b.boxes = boxes
	if b.list != nil {
		b.list.Refresh()
	}
}
