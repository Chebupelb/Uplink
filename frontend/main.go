package main

import (
	"syscall/js"
)

type App struct {
	doc  js.Value
	root js.Value
}

func main() {
	app := &App{
		doc: js.Global().Get("document"),
	}
	app.root = app.doc.Call("getElementById", "app")

	js.Global().Set("pressSubmit", js.FuncOf(func(this js.Value, args []js.Value) any {
		handleAuth(app, args[0].Bool())
		return nil
	}))
	js.Global().Set("switchView", js.FuncOf(func(this js.Value, args []js.Value) any {
		renderAuth(app, args[0].Bool())
		return nil
	}))
	js.Global().Set("logout", js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("localStorage").Call("removeItem", "token")
		app.navigate("/")
		return nil
	}))
	js.Global().Set("changeTab", js.FuncOf(func(this js.Value, args []js.Value) any {
		renderMenu(app, args[0].String())
		return nil
	}))

	app.router()
	select {}
}

func (a *App) router() {
	p := js.Global().Get("location").Get("pathname").String()
	t := js.Global().Get("localStorage").Call("getItem", "token")
	if t.IsNull() && p != "/" { a.navigate("/"); return }
	if !t.IsNull() && p == "/" { a.navigate("/menu"); return }
	if p == "/" { renderAuth(a, false) } else { renderMenu(a, "dashboard") }
}

func (a *App) navigate(path string) {
	js.Global().Get("history").Call("pushState", nil, "", path)
	a.router()
}