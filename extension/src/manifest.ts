import pkg from "../package.json";

export const manifest: chrome.runtime.ManifestV3 = {
  author: pkg.author,
  description: pkg.description,
  name: pkg.displayName ?? pkg.name,
  version: pkg.version,
  homepage_url: pkg.homepage,
  manifest_version: 3,
  action: {
    default_icon: {
      48: "icons/48.png",
    },
    default_popup: "src/popup.html#popup",
    default_title: "Open Terminal",
  },
  background: {
    service_worker: "src/background.ts",
  },
  omnibox: {
    keyword: "tty",
  },
  commands: {
    "_execute_action": {
      description: "Show terminal popup",
      suggested_key: {
        default: "Ctrl+E",
        mac: "Command+E",
      }
    },
    "open-terminal-tab": {
      description: "Create a new terminal tab",
      suggested_key: {
        default: "Ctrl+Shift+E",
        mac: "Command+Shift+E",
      }
    }
  },
  permissions: [
    "nativeMessaging",
    "tabs",
    "history",
    "clipboardWrite",
    "offscreen",
    "system.display",
    "bookmarks",
    "storage",
    "downloads",
    "sidePanel",
    "contextMenus",
    "management",
    "scripting",
  ],
  host_permissions: ["*://*/*"],
  icons: {
    16: "icons/16.png",
    48: "icons/48.png",
    128: "icons/128.png",
    256: "icons/256.png",
    512: "icons/512.png",
  },
};
