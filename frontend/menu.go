package main

func renderMenu(a *App, tab string) {
	act := "bg-[#00f3ff] text-black shadow-[0_0_15px_#00f3ff]"
	inact := "hover:bg-[#00f3ff]/10 border border-transparent hover:border-[#00f3ff]/30"
	ds, ls, hs := inact, inact, inact
	cont := ""

	if tab == "dashboard" {
		ds = act
		cont = `
			<div class="grid grid-cols-2 gap-4">
				<div class="hud-border p-6 bg-[#00f3ff]/5">
					<div class="text-[10px] opacity-40 mb-1">RANK</div>
					<div class="text-4xl font-bold tracking-tighter">#000</div>
				</div>
				<div class="hud-border p-6 bg-[#00f3ff]/5">
					<div class="text-[10px] opacity-40 mb-1">SPEED</div>
					<div class="text-4xl font-bold tracking-tighter">0.0<span class="text-sm opacity-30 ml-2">WPM</span></div>
				</div>
			</div>
			<div class="mt-8 hud-border p-10 text-center opacity-20 border-dashed text-xs tracking-widest">
				NO_DATA_LINK_FOUND
			</div>`
	} else {
		if tab == "leaderboard" { ls = act } else { hs = act }
		cont = `<div class="opacity-40 tracking-[0.5em] text-center mt-20 text-xs">SCANNING_LOCAL_NET...</div>`
	}

	html := `
	<div class="fixed inset-0 flex">
		<div class="w-72 border-r border-[#00f3ff]/20 bg-black/40 backdrop-blur-xl p-8 flex flex-col">
			<div class="mb-20">
				<div class="text-3xl font-bold glow-text tracking-tighter" style="font-family: 'Orbitron';">UP<span class="opacity-20">LINK</span></div>
				<div class="text-[8px] tracking-[0.4em] opacity-30 mt-1">OS_VER_2.4</div>
			</div>
			<nav class="flex-1 space-y-4">
				<button onclick="changeTab('dashboard')" class="w-full p-4 text-[10px] text-left tracking-[0.3em] font-bold transition-all `+ds+` ">DASHBOARD</button>
				<button onclick="changeTab('leaderboard')" class="w-full p-4 text-[10px] text-left tracking-[0.3em] font-bold transition-all `+ls+` ">LEADERBOARD</button>
				<button onclick="changeTab('history')" class="w-full p-4 text-[10px] text-left tracking-[0.3em] font-bold transition-all `+hs+` ">LOGS</button>
			</nav>
			<button onclick="logout()" class="border border-red-500/30 text-red-500/50 p-2 text-[9px] tracking-[0.5em] hover:bg-red-500/10">DISCONNECT</button>
		</div>
		<div class="flex-1 p-12 overflow-y-auto relative">
			<header class="flex justify-between items-end mb-12 border-b border-[#00f3ff]/10 pb-4 uppercase">
				<div>
					<div class="text-[9px] opacity-30 tracking-[0.4em]">PATH</div>
					<div class="text-xl font-bold tracking-[0.2em]">ROOT/`+tab+`</div>
				</div>
				<div class="text-[9px] text-[#00f3ff] border border-[#00f3ff] px-3 py-1 mb-1 font-bold tracking-widest">LINK_OK</div>
			</header>
			`+cont+`
		</div>
	</div>`
	a.root.Set("innerHTML", html)
}