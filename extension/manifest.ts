export default {
    key: "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAlT6IZrhsX6Y4GNar3eVY7gDmH1mujxci6QeeaiFMvz4TrJ1VSCy4eCBC3XLBnmN4Evi//pgG8XeW9Ock4aZhLr/zQboQ8uuqv2V/MvHYRZqirghYnVIPZ8FaMiYIwCPpG/dB+PsYlpsxtb0vDEfa0RYt7uUAERBhOCIX/j47TdiuUpvARKZaoPSFZCUdgq7n4XcEv0sZtjhuXR2tD7rgqmZgu6FGlO4CvshdWcXHMmiZWssfYcHUGeJP/Zbcs0tqwk7LstT80zGtVSUu1ey7CQxKZTAaNZVglyye2rSECR52UzTIeHI92gZjsFl7tENs3Hs+lY3ReVJGhhF3ksTn0QIDAQAB",
    name: "Tweety",
    description: "An integrated terminal for your browser",
    version: "0.1.0",
    manifest_version: 3,
    side_panel: {
        default_path: "term.html",
    },
    devtools_page: "src/devtools.html",
    commands: {
        _execute_action: {
            suggested_key: {
                default: "Ctrl+J",
                mac: "Command+J",
            }
        },
        openInNewTab: {
            description: "Open in new tab",
            suggested_key: {
                default: "Ctrl+Shift+T",
                mac: "Command+Shift+T",
            }
        },
        openInNewWindow: {
            description: "Open in new window",
        },
    },
    action: {
        default_icon: {
            16: "icons/icon16.png",
            32: "icons/icon32.png",
            48: "icons/icon48.png",
        },
        default_title: "Create Terminal",
        default_popup: "src/popup.html",
    },
    background: {
        service_worker: "src/service_worker.ts",
    },
    permissions: [
        "nativeMessaging",
        "tabs",
        "notifications",
        "history",
        "contextMenus",
        "bookmarks",
        "storage",
        "sidePanel",
        "scripting"
    ],
    host_permissions: ["<all_urls>"],
    icons: {
        16: "icons/icon16.png",
        32: "icons/icon32.png",
        48: "icons/icon48.png",
        128: "icons/icon128.png",
    },
} satisfies chrome.runtime.ManifestV3
