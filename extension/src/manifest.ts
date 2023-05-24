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
    default_popup: "src/entries/popup/index.html",
  },
  background: {
    service_worker: "src/entries/background/main.ts",
  },
  chrome_url_overrides: {
    newtab: "src/entries/popup/index.html",
  },
  permissions: [
    "nativeMessaging",
    "tabs",
    "history",
    "bookmarks",
    "downloads",
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
