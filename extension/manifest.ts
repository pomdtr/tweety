export default {
    author: {
        email: "contact@pomdtr.me",
    },
    name: "tweety",
    version: "0.1.0",
    manifest_version: 3,
    commands: {
        openInNewTab: {
            description: "Open in new tab",
        },
        openInNewWindow: {
            description: "Open in new window",
        },
        openinPopupWindow: {
            description: "Open in popup window",
        }
    },
    action: {
        default_icon: {
            16: "icons/16.png",
            19: "icons/19.png",
            32: "icons/32.png",
            38: "icons/38.png",
        },
        default_title: "Create Terminal",
        default_popup: "src/popup.html",
    },
    background: {
        service_worker: "src/worker.ts",
    },
    permissions: [
        "nativeMessaging",
        "tabs",
        "history",
        "contextMenus",
        "bookmarks",
    ],
    host_permissions: ["<all_urls>"],
    icons: {
        16: "icons/16.png",
        19: "icons/19.png",
        32: "icons/32.png",
        38: "icons/38.png",
        48: "icons/48.png",
        64: "icons/64.png",
        96: "icons/96.png",
        128: "icons/128.png",
        256: "icons/256.png",
        512: "icons/512.png",
    },
} satisfies chrome.runtime.ManifestV3
