import degit from "degit"
import os from "os"
import path from "path"
import fs from "fs/promises"

const dirname = path.dirname(new URL(import.meta.url).pathname)
const emitter = await degit('mbadolato/iTerm2-Color-Schemes/vscode', {
    cache: true,
    force: true,
    verbose: true
})

emitter.on('info', info => {
    console.log(info.message)
})

const cloneDir = path.join(os.tmpdir(), "degit")
await emitter.clone(cloneDir)

const entries = await fs.readdir(cloneDir, { withFileTypes: true })
const keyMapping = {
    "terminal.foreground": "foreground",
    "terminal.background": "background",
    "terminal.ansiBlack": "ansiBlack",
    "terminal.ansiBlue": "ansiBlue",
    "terminal.ansiCyan": "ansiCyan",
    "terminal.ansiGreen": "ansiGreen",
    "terminal.ansiMagenta": "ansiMagenta",
    "terminal.ansiRed": "ansiRed",
    "terminal.ansiWhite": "ansiWhite",
    "terminal.ansiYellow": "ansiYellow",
    "terminal.ansiBrightBlack": "ansiBrightBlack",
    "terminal.ansiBrightBlue": "ansiBrightBlue",
    "terminal.ansiBrightCyan": "ansiBrightCyan",
    "terminal.ansiBrightGreen": "ansiBrightGreen",
    "terminal.ansiBrightMagenta": "ansiBrightMagenta",
    "terminal.ansiBrightRed": "ansiBrightRed",
    "terminal.ansiBrightWhite": "ansiBrightWhite",
    "terminal.ansiBrightYellow": "ansiBrightYellow",
    "terminal.selectionBackground": "selectionBackground",
    "terminalCursor.foreground": "cursor"
}
for (const entry of entries) {
    const vscodeTheme = JSON.parse(await fs.readFile(path.join(cloneDir, entry.name), { encoding: "utf-8" }))
    const xtermTheme = {}
    for (const [key, value] of Object.entries(vscodeTheme["workbench.colorCustomizations"])) {
        xtermTheme[keyMapping[key]] = value
    }
    await fs.writeFile(path.join(dirname, "..", "src", "themes", entry.name), JSON.stringify(xtermTheme, null, 4))
}

await fs.rm(path.join(cloneDir), { recursive: true, force: true })
