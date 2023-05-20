import pkg from "../package.json";

export const manifest: chrome.runtime.ManifestV3 = {
  author: pkg.author,
  description: pkg.description,
  name: pkg.displayName ?? pkg.name,
  version: pkg.version,
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
  permissions: ["nativeMessaging", "tabs", "history", "bookmarks", "downloads"],
  host_permissions: ["*://*/*"],
  icons: {
    48: "icons/48.png",
    128: "icons/128.png",
    256: "icons/256.png",
    512: "icons/512.png",
  },
};
