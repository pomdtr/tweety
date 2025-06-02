import * as fs from "fs"

fs.rmSync("dist/firefox", { recursive: true, force: true })
fs.cpSync("dist/chrome", "dist/firefox", { recursive: true })

const manifest = JSON.parse(fs.readFileSync("dist/chrome/manifest.json", "utf-8"))

delete manifest.side_panel
manifest.permissions = manifest.permissions.filter((p: string) => p !== "sidePanel")
manifest.browser_specific_settings = {
    gecko: {
        id: "tweety@pomdtr.me"
    }
}

fs.writeFileSync("dist/firefox/manifest.json", JSON.stringify(manifest, null, 2))
