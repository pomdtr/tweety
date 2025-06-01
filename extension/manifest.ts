export default {
    author: {
        email: "contact@pomdtr.me",
    },
    name: "tweety",
    version: "0.1.0",
    manifest_version: 3,
    omnibox: { keyword: "tty" },
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
            16: "icons/icon16.png",
            32: "icons/icon32.png",
            48: "icons/icon48.png",
        },
        default_title: "Create Terminal",
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
        "storage"
    ],
    host_permissions: ["<all_urls>"],
    icons: {
        16: "icons/icon16.png",
        32: "icons/icon32.png",
        48: "icons/icon48.png",
        128: "icons/icon128.png",
    },
} satisfies chrome.runtime.ManifestV3
