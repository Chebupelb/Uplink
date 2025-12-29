package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
)

func (a *App) showErrorModal(message string) {
	overlay := a.doc.Call("createElement", "div")
	overlay.Set("className", "fixed inset-0 flex items-center justify-center bg-black/80 backdrop-blur-md z-[100] animate-in fade-in duration-300")
	overlay.Set("id", "error-overlay")

	overlay.Set("innerHTML", `
		<div class="hud-border bg-black p-8 max-w-sm w-full mx-4 border-red-500/50 shadow-[0_0_30px_rgba(239,68,68,0.2)] text-center">
			<div class="text-red-500 text-[10px] tracking-[0.5em] mb-2">ACCESS_DENIED</div>
			<h2 class="text-white text-xl font-bold mb-4 uppercase tracking-wider">Лимит достигнут</h2>
			<p class="text-gray-400 text-sm mb-6 normal-case font-sans">`+message+`</p>
			<button id="close-error-btn" class="w-full hud-border py-3 border-red-500/50 text-red-500 hover:bg-red-500/10 transition-all uppercase text-xs tracking-widest font-bold">
				Вернуться в терминал
			</button>
		</div>
	`)

	a.doc.Get("body").Call("appendChild", overlay)

	a.doc.Call("getElementById", "close-error-btn").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
		overlay.Call("remove")
		a.navigate("/menu")
		return nil
	}))
}

func (a *App) renderLobbyPage(roomID string) {
	a.CurrentRoomID = roomID

	html := `
    <div class="fixed inset-0 flex flex-col bg-transparent text-[#00f3ff] font-mono uppercase overflow-hidden">
        <header class="p-6 border-b border-[#00f3ff]/20 bg-black/40 backdrop-blur-md flex justify-between items-center">
            <div>
                <div class="text-[10px] opacity-40 tracking-[0.5em]">INSTANCE_ID</div>
                <div class="text-2xl font-bold text-white shadow-[#00f3ff] drop-shadow-md">` + roomID + `</div>
            </div>
            <div class="flex gap-6">
                <button id="exit-btn" class="hud-border px-4 py-2 hover:bg-red-500/20 border-red-500/50 text-red-500 text-xs transition-all">
                    ABORT_SESSION
                </button>
            </div>
        </header>

        <main class="flex-1 flex p-6 gap-6 overflow-hidden">
            <div class="w-64 flex flex-col gap-4">
                <div id="host-settings" class="hud-border bg-black/60 p-4 hidden">
                    <h3 class="text-[10px] mb-4 tracking-[0.3em] border-b border-[#00f3ff]/20 pb-2">SESSION_CONFIG</h3>
                    <div class="flex flex-col gap-4">
                        <div class="flex flex-col gap-1">
                            <label class="text-[9px] opacity-60">MAX_NETRUNERS</label>
                            <select id="max-players-select" class="bg-black border border-[#00f3ff]/30 text-[#00f3ff] p-1 text-xs focus:outline-none focus:border-[#00f3ff]">
                                <option value="1">1 (SOLO)</option>
                                <option value="2">2 (DUEL)</option>
                                <option value="3">3 (TRIO)</option>
                                <option value="4">4 (SQUAD)</option>
                                <option value="8">8 (RAID)</option>
                            </select>
                        </div>
                        <div class="flex flex-col gap-1">
                            <label class="text-[9px] opacity-60">LANGUAGE</label>
                            <select id="language-select" class="bg-black border border-[#00f3ff]/30 text-[#00f3ff] p-1 text-xs focus:outline-none focus:border-[#00f3ff]">
                                <option value="ru">RUSSIAN (RU)</option>
                                <option value="en">ENGLISH (EN)</option>
                            </select>
                        </div>
                        <div class="flex flex-col gap-1">
                            <label class="text-[9px] opacity-60">CATEGORY</label>
                            <select id="category-select" class="bg-black border border-[#00f3ff]/30 text-[#00f3ff] p-1 text-xs focus:outline-none focus:border-[#00f3ff]">
                                <option value="">LOADING...</option>
                            </select>
                        </div>
                    </div>
                </div>

                <div class="hud-border bg-black/60 p-4 flex-1 overflow-y-auto">
                    <h3 class="text-[10px] mb-4 tracking-[0.3em] border-b border-[#00f3ff]/20 pb-2">CONNECTED_NETRUNERS</h3>
                    <div id="player-list" class="space-y-3 font-sans normal-case"></div>
                </div>
                
                <button id="start-btn" style="display: none;" class="bg-[#00f3ff] text-black py-4 font-bold hover:bg-white transition-all tracking-[.3em] text-sm shadow-[0_0_15px_rgba(0,243,255,0.5)]">
                    START_UPLINK
                </button>
            </div>

            <div class="flex-1 flex flex-col hud-border bg-black/40 backdrop-blur-sm overflow-hidden">
                <div class="p-2 border-b border-[#00f3ff]/10 text-[10px] opacity-50">SECURE_CHANNEL_v4.2</div>
                <div id="chat-messages" class="flex-1 p-4 overflow-y-auto space-y-2 text-sm normal-case font-sans"></div>
                <div class="p-4 border-t border-[#00f3ff]/20 bg-black/20 flex gap-2">
                    <input id="chat-input" type="text" placeholder="TYPE MESSAGE..." 
                        class="flex-1 bg-transparent border border-[#00f3ff]/30 p-2 text-[#00f3ff] focus:outline-none focus:border-[#00f3ff] placeholder:opacity-30">
                    <button id="chat-send" class="px-4 py-2 bg-[#00f3ff]/10 border border-[#00f3ff]/50 hover:bg-[#00f3ff]/30">SEND</button>
                </div>
            </div>
        </main>
    </div>`

	a.root.Set("innerHTML", html)

	

	sendSettings := func() {
		if a.Socket.IsUndefined() || a.Socket.IsNull() {
			return
		}
		maxVal, _ := strconv.Atoi(a.doc.Call("getElementById", "max-players-select").Get("value").String())
		langVal := a.doc.Call("getElementById", "language-select").Get("value").String()
		catVal := a.doc.Call("getElementById", "category-select").Get("value").String()

		msg := map[string]any{
			"type": "update_settings",
			"payload": map[string]any{
				"max_players": maxVal,
				"language":    langVal,
				"category":    catVal,
			},
		}
		data, _ := json.Marshal(msg)
		a.Socket.Call("send", string(data))
	}

	sendChat := func() {
		el := a.doc.Call("getElementById", "chat-input")
		val := el.Get("value").String()
		if val == "" || a.Socket.IsUndefined() || a.Socket.IsNull() {
			return
		}

		msg := map[string]any{
			"type": "chat_message",
			"payload": map[string]string{
				"text": val,
			},
		}
		data, _ := json.Marshal(msg)
		a.Socket.Call("send", string(data))
		el.Set("value", "")
	}

	for _, id := range []string{"max-players-select", "language-select", "category-select"} {
		a.doc.Call("getElementById", id).Set("onchange", js.FuncOf(func(this js.Value, args []js.Value) any {
			sendSettings()
			return nil
		}))
	}

	a.doc.Call("getElementById", "exit-btn").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
		a.navigate("/menu")
		return nil
	}))

	a.doc.Call("getElementById", "start-btn").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
		msg := map[string]any{"type": "game_start"}
		data, _ := json.Marshal(msg)
		a.Socket.Call("send", string(data))
		return nil
	}))

	a.doc.Call("getElementById", "chat-send").Set("onclick", js.FuncOf(func(this js.Value, args []js.Value) any {
		sendChat()
		return nil
	}))

	a.doc.Call("getElementById", "chat-input").Set("onkeypress", js.FuncOf(func(this js.Value, args []js.Value) any {
		if args[0].Get("key").String() == "Enter" {
			sendChat()
		}
		return nil
	}))

	a.loadCategories()
	go func() { a.setupLobbyWS(roomID) }()
}

func (a *App) loadCategories() {
	go func() {
		resp, err := http.Get("/api/v1/categories")
		if err != nil {
			return
		}
		defer resp.Body.Close()

		var res struct {
			Categories []string `json:"categories"`
		}
		json.NewDecoder(resp.Body).Decode(&res)

		selectEl := a.doc.Call("getElementById", "category-select")
		html := ""
		for _, cat := range res.Categories {
			html += fmt.Sprintf(`<option value="%s">%s</option>`, cat, strings.ToUpper(cat))
		}
		selectEl.Set("innerHTML", html)
	}()
}

func (a *App) setupLobbyWS(roomID string) js.Value {
	protocol := "ws://"
	if js.Global().Get("location").Get("protocol").String() == "https:" {
		protocol = "wss://"
	}

	ls := js.Global().Get("localStorage")
	tokenVal := ls.Call("getItem", "auth_token")
	if tokenVal.IsNull() {
		tokenVal = ls.Call("getItem", "token")
	}
	token := tokenVal.String()

	url := fmt.Sprintf("%s%s/ws?room_id=%s&token=%s", protocol, js.Global().Get("location").Get("host").String(), roomID, token)
	ws := js.Global().Get("WebSocket").New(url)

	a.Socket = ws

	ws.Set("onclose", js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		if event.Get("reason").String() == "LOBBY_FULL" || event.Get("code").Int() == 4008 {
			a.showErrorModal("В данной сессии достигнут максимальный лимит агентов.")
		}
		return nil
	}))

	ws.Set("onmessage", js.FuncOf(func(this js.Value, args []js.Value) any {
		jsonData := args[0].Get("data").String()
		var rawMsg struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}
		json.Unmarshal([]byte(jsonData), &rawMsg)

		switch rawMsg.Type {
		case "player_joined", "lobby_update":
			a.updateAgentsUI(rawMsg.Payload)

		case "update_settings":
			var settings struct {
				MaxPlayers int    `json:"max_players"`
				Language   string `json:"language"`
				Category   string `json:"category"`
			}
			if err := json.Unmarshal(rawMsg.Payload, &settings); err == nil {
				a.syncSelectValue("max-players-select", strconv.Itoa(settings.MaxPlayers))
				a.syncSelectValue("language-select", settings.Language)
				a.syncSelectValue("category-select", settings.Category)
			}

		case "game_start":
			a.renderGamePage(rawMsg.Payload)

		case "chat_message":
			var chatData struct {
				Sender string `json:"sender_name"`
				Text   string `json:"text"`
			}
			if err := json.Unmarshal(rawMsg.Payload, &chatData); err == nil {
				container := a.doc.Call("getElementById", "chat-messages")
				if !container.IsNull() {
					msgHtml := fmt.Sprintf(`
                        <div class="mb-2 animate-in fade-in slide-in-from-left-2 duration-300">
                            <span class="text-[#00f3ff] font-bold text-[10px] mr-2">[%s]:</span>
                            <span class="text-white/90 text-sm">%s</span>
                        </div>
                    `, chatData.Sender, chatData.Text)
					container.Call("insertAdjacentHTML", "beforeend", msgHtml)
					container.Set("scrollTop", container.Get("scrollHeight"))
				}
			}
		}
		return nil
	}))

	return ws
}

func (a *App) syncSelectValue(id, val string) {
	el := a.doc.Call("getElementById", id)
	if !el.IsNull() && !a.doc.Get("activeElement").Equal(el) {
		el.Set("value", val)
	}
}

func (a *App) updateAgentsUI(payload json.RawMessage) {
	var players []struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
		IsOwner  bool   `json:"is_owner"`
	}
	json.Unmarshal(payload, &players)

	playerListEl := a.doc.Call("getElementById", "player-list")
	isImOwner := false
	html := ""
	for _, p := range players {
		if a.User != nil && p.UserID == a.User.ID && p.IsOwner {
			isImOwner = true
		}
		badge := ""
		if p.IsOwner {
			badge = ` <span class="text-[9px] border border-[#00f3ff] px-1 text-[#00f3ff]">HOST</span>`
		}
		html += fmt.Sprintf(`<div class="flex items-center gap-2 py-1"><div class="w-1.5 h-1.5 bg-[#00f3ff]"></div><div class="text-sm">%s%s</div></div>`, p.Username, badge)
	}
	if len(players) == 1 {
		isImOwner = true
	}
	playerListEl.Set("innerHTML", html)

	if el := a.doc.Call("getElementById", "host-settings"); !el.IsNull() {
		el.Get("style").Set("display", "block")
	}

	for _, id := range []string{"max-players-select", "language-select", "category-select", "start-btn"} {
		el := a.doc.Call("getElementById", id)
		if el.IsNull() {
			continue
		}
		if id == "start-btn" {
			if isImOwner {
				el.Get("style").Set("display", "block")
			} else {
				el.Get("style").Set("display", "none")
			}
		} else {
			el.Set("disabled", !isImOwner)
		}
	}
}