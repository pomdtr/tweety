export default {
    name: "Tweety",
    description: "An integrated terminal for your browser",
    version: "0.1.0",
    manifest_version: 3,
    side_panel: {
        default_path: "term.html",
    },
    devtools_page: "src/devtools.html",
    commands: {
        openInNewTab: {
            description: "Open in new tab",
            suggested_key: {
                linux: "Ctrl+Shift+T",
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
    },
    background: {
        service_worker: "src/service_worker.ts",
        // @ts-ignore
        scripts: ["src/service_worker.ts"],
    },
    permissions: [
        "nativeMessaging",
        "tabs",
        "notifications",
        "history",
        "contextMenus",
        "bookmarks",
        "storage",
        "sidePanel"
    ],
    host_permissions: ["<all_urls>"],
    icons: {
        16: "icons/icon16.png",
        32: "icons/icon32.png",
        48: "icons/icon48.png",
        128: "icons/icon128.png",
    },
} satisfies chrome.runtime.ManifestV3
