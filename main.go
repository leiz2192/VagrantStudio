package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/leiz2192/vagrantstudio/internal"
)

var about = `
Program Name: Vagrant Studio
Version: 0.1
Author: leiz2192
Description: This is a simple vagrant gui program.
License: MIT`

func main() {
	myApp := app.NewWithID("com.leiz2192.vagrantstudio")
	myApp.Settings().SetTheme(&internal.CustomTheme{})
	myWindow := myApp.NewWindow("Vagrant Studio")

	box := internal.NewBox()
	env, err := internal.NewEnvironment()
	if err != nil {
		dialog.ShowError(err, myWindow)
		return
	}

	tabs := container.NewAppTabs(
		container.NewTabItem("Env", env.NewContent()),
		container.NewTabItem("Box", box.NewContent()),
		container.NewTabItem("About", widget.NewLabel(about)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	myWindow.SetContent(tabs)

	myWindow.SetCloseIntercept(func() {
		if err := env.Close(); err != nil {
			dialog.ShowError(err, myWindow)
			return
		}
		myWindow.Close()
	})

	myWindow.Resize(fyne.NewSize(800, 600))
	myWindow.CenterOnScreen()
	myWindow.ShowAndRun()
}
