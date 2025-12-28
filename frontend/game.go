package main

import (
	"encoding/json"
	"fmt"
	"math"
	"syscall/js"
	"time"
)

type GameState struct {
	FullText     []rune
	CurrentIndex int
	Errors       int
	StartTime    time.Time
	IsFinished   bool
	WPM          int
	Accuracy     float64
}

var game *GameState

func (a *App) renderGamePage(payload json.RawMessage) {
	var startData struct {
		Text      string    `json:"text"`
		StartTime time.Time `json:"start_time"`
	}
	json.Unmarshal(payload, &startData)

	game = &GameState{
		FullText:     []rune(startData.Text),
		StartTime:    startData.StartTime,
		CurrentIndex: 0,
		Accuracy:     100.0,
	}

	html := `
    <div class="fixed inset-0 flex flex-col bg-black text-[#00f3ff] font-mono select-none overflow-hidden">
        <div class="flex justify-between items-end p-6 border-b border-[#00f3ff]/20 bg-black/80 backdrop-blur">
            <div>
                <div class="text-[10px] opacity-40 tracking-[0.5em] mb-1">LIVE_FEED</div>
                <div class="text-2xl font-bold glow-text">UPLINK_ESTABLISHED</div>
            </div>
            <div class="flex gap-12 text-center">
                <div>
                    <div class="text-[9px] opacity-40 tracking-widest">SPEED (WPM)</div>
                    <div id="hud-wpm" class="text-4xl font-bold font-mono">0</div>
                </div>
                <div>
                    <div class="text-[9px] opacity-40 tracking-widest">ACCURACY</div>
                    <div id="hud-acc" class="text-4xl font-bold font-mono">100%</div>
                </div>
            </div>
        </div>

        <div class="flex-1 flex flex-col items-center justify-center relative">
            <div id="countdown-overlay" class="absolute inset-0 flex items-center justify-center z-50 bg-black/90">
                <div id="countdown-text" class="text-9xl font-bold text-[#00f3ff] animate-pulse">3</div>
            </div>

            <div class="max-w-4xl w-full p-8 relative z-10">
                <div id="game-text" class="text-2xl md:text-4xl leading-relaxed tracking-wide font-medium font-mono break-words outline-none text-center">
                </div>
            </div>
        </div>

        <div class="p-6 border-t border-[#00f3ff]/20 bg-black/80 backdrop-blur h-48 overflow-y-auto">
            <div class="text-[9px] opacity-40 tracking-[0.3em] mb-4">NETWORK_ACTIVITY</div>
            <div id="opponents-container" class="space-y-4">
            </div>
        </div>
    </div>`

	a.root.Set("innerHTML", html)
	a.renderGameText()
	a.startCountdown()

	keydownHandler := js.FuncOf(func(this js.Value, args []js.Value) any {
		if game.IsFinished || time.Now().Before(game.StartTime) {
			return nil
		}
		event := args[0]
		key := event.Get("key").String()

		if len([]rune(key)) != 1 {
			return nil
		}

		event.Call("preventDefault")
		a.handleTyping(key)
		return nil
	})

	js.Global().Get("window").Set("onkeydown", keydownHandler)

	if !a.Socket.IsUndefined() && !a.Socket.IsNull() {
		a.Socket.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) any {
			jsonData := args[0].Get("data").String()
			var msg struct {
				Type    string          `json:"type"`
				Payload json.RawMessage `json:"payload"`
			}
			json.Unmarshal([]byte(jsonData), &msg)

			switch msg.Type {
			case "state_update":
				a.updateOpponentsUI(msg.Payload)
			case "game_end":
				game.IsFinished = true
				js.Global().Get("window").Set("onkeydown", nil)
				a.showResultsModal(msg.Payload)
			}
			return nil
		}))
	}
}

func (a *App) startCountdown() {
	go func() {
		overlay := a.doc.Call("getElementById", "countdown-overlay")
		text := a.doc.Call("getElementById", "countdown-text")
		for {
			remaining := time.Until(game.StartTime)
			if remaining <= 0 {
				break
			}
			seconds := int(math.Ceil(remaining.Seconds()))
			if !text.IsNull() {
				text.Set("innerText", fmt.Sprintf("%d", seconds))
			}
			time.Sleep(100 * time.Millisecond)
		}
		if !text.IsNull() {
			text.Set("innerText", "UPLOAD!")
			text.Get("classList").Call("add", "text-red-500")
		}
		time.Sleep(400 * time.Millisecond)
		if !overlay.IsNull() {
			overlay.Call("remove")
		}
	}()
}

func (a *App) renderGameText() {
	el := a.doc.Call("getElementById", "game-text")
	if el.IsNull() {
		return
	}

	passed := string(game.FullText[:game.CurrentIndex])
	current := ""
	if game.CurrentIndex < len(game.FullText) {
		current = string(game.FullText[game.CurrentIndex])
	}
	future := ""
	if game.CurrentIndex+1 < len(game.FullText) {
		future = string(game.FullText[game.CurrentIndex+1:])
	}

	html := fmt.Sprintf(`<span class="text-[#00f3ff] shadow-[0_0_10px_#00f3ff] whitespace-pre-wrap">%s</span>`, passed)
	if current != "" {
		displayChar := current
		if current == " " {
			displayChar = "&nbsp;"
		}
		html += fmt.Sprintf(`<span id="cursor-char" class="bg-[#00f3ff] text-black px-0.5 mx-px animate-pulse whitespace-pre-wrap">%s</span>`, displayChar)
	}
	html += fmt.Sprintf(`<span class="text-white/10 whitespace-pre-wrap">%s</span>`, future)

	el.Set("innerHTML", html)
}

func (a *App) handleTyping(key string) {
	targetChar := string(game.FullText[game.CurrentIndex])

	if key == targetChar {
		game.CurrentIndex++
		a.updateStats()

		msg := map[string]any{
			"type": "client_input",
			"payload": map[string]any{
				"current_index": game.CurrentIndex,
				"wpm":           game.WPM,
				"accuracy":      int(game.Accuracy),
			},
		}
		data, _ := json.Marshal(msg)
		a.Socket.Call("send", string(data))

		if game.CurrentIndex >= len(game.FullText) {
			game.IsFinished = true
		}
	} else {
		game.Errors++
		a.updateStats()
		flash := a.doc.Call("createElement", "div")
		flash.Set("className", "fixed inset-0 bg-red-500/10 pointer-events-none z-[60]")
		a.doc.Get("body").Call("appendChild", flash)
		time.AfterFunc(80*time.Millisecond, func() { flash.Call("remove") })
	}

	a.renderGameText()
}

func (a *App) updateStats() {
	elapsed := time.Since(game.StartTime).Minutes()
	if elapsed > 0 {
		game.WPM = int((float64(game.CurrentIndex) / 5.0) / elapsed)
	}
	totalPresses := game.CurrentIndex + game.Errors
	if totalPresses > 0 {
		game.Accuracy = (float64(game.CurrentIndex) / float64(totalPresses)) * 100
	}

	wpmEl := a.doc.Call("getElementById", "hud-wpm")
	if !wpmEl.IsNull() {
		wpmEl.Set("innerText", fmt.Sprintf("%d", game.WPM))
	}

	accEl := a.doc.Call("getElementById", "hud-acc")
	if !accEl.IsNull() {
		accEl.Set("innerText", fmt.Sprintf("%.0f%%", game.Accuracy))
	}
}

func (a *App) updateOpponentsUI(payload json.RawMessage) {
	var states []struct {
		UserID   string  `json:"user_id"`
		Username string  `json:"username"`
		Progress float64 `json:"progress"`
		WPM      int     `json:"wpm"`
	}
	json.Unmarshal(payload, &states)
	container := a.doc.Call("getElementById", "opponents-container")
	if container.IsNull() {
		return
	}

	html := ""
	totalChars := float64(len(game.FullText))
	if totalChars == 0 {
		totalChars = 1
	}

	for _, s := range states {
		if a.User != nil && s.UserID == a.User.ID {
			continue
		}
		percent := (s.Progress / totalChars) * 100
		name := s.Username
		if name == "" {
			name = "AGENT_" + s.UserID[:4]
		}
		html += fmt.Sprintf(`
        <div class="mb-3">
            <div class="flex justify-between text-[10px] font-mono mb-1 text-[#00f3ff]">
                <span>%s</span>
                <span class="opacity-50">%d WPM</span>
            </div>
            <div class="h-1 w-full bg-[#00f3ff]/10">
                <div class="h-full bg-[#00f3ff] shadow-[0_0_8px_#00f3ff] transition-all duration-300" style="width: %.1f%%;"></div>
            </div>
        </div>`, name, s.WPM, percent)
	}
	container.Set("innerHTML", html)
}

func (a *App) showResultsModal(payload json.RawMessage) {
	var res struct {
		Results []struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			WPM      int    `json:"wpm"`
			Accuracy int    `json:"accuracy"`
		} `json:"results"`
	}
	json.Unmarshal(payload, &res)

	overlay := a.doc.Call("createElement", "div")
	overlay.Set("className", "fixed inset-0 flex items-center justify-center bg-black/95 backdrop-blur-md z-[200]")

	rowsHtml := ""
	for i, r := range res.Results {
		rankColor := "#00f3ff"
		if i == 0 {
			rankColor = "#ffd700"
		}
		name := r.Username
		if name == "" {
			name = "AGENT_" + r.UserID[:4]
		}

		rowsHtml += fmt.Sprintf(`
			<div class="flex items-center justify-between p-4 border-b border-[#00f3ff]/10 bg-[#00f3ff]/5 mb-2">
				<div class="flex items-center gap-4">
					<span class="text-2xl font-black" style="color: %s">#%d</span>
					<div class="text-white font-bold">%s</div>
				</div>
				<div class="flex gap-8 text-right font-mono">
					<div><div class="text-[8px] opacity-40">WPM</div><div class="text-xl text-[#00f3ff]">%d</div></div>
					<div><div class="text-[8px] opacity-40">ACC</div><div class="text-xl text-white">%d%%</div></div>
				</div>
			</div>`, rankColor, i+1, name, r.WPM, r.Accuracy)
	}

	overlay.Set("innerHTML", `
		<div class="max-w-xl w-full mx-4 p-1 bg-[#00f3ff]/20">
			<div class="bg-black p-8 border border-[#00f3ff]/50 shadow-[0_0_50px_rgba(0,243,255,0.2)]">
				<div class="text-center mb-8">
					<div class="text-[#00f3ff] text-[10px] tracking-[0.8em] mb-2 uppercase">Sync_Complete</div>
					<h2 class="text-3xl font-black text-white italic uppercase tracking-tighter">SESSION RESULTS</h2>
				</div>
				<div class="space-y-1 mb-8">`+rowsHtml+`</div>
				<div class="flex">
					<button id="res-menu" class="w-full py-4 bg-red-500/10 border border-red-500/50 text-red-500 hover:bg-red-500/20 transition-all uppercase text-xs font-bold tracking-[0.2em]">ВЕРНУТЬСЯ В ТЕРМИНАЛ</button>
				</div>
			</div>
		</div>`)

	a.doc.Get("body").Call("appendChild", overlay)

	a.doc.Call("getElementById", "res-menu").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
		overlay.Call("remove")
		if !a.Socket.IsUndefined() && !a.Socket.IsNull() {
			a.Socket.Call("close")
		}
		a.navigate("/menu")
		return nil
	}))
}