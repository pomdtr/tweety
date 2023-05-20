import browser from "webextension-polyfill";

type Message = {
  id: string;
  payload: any;
  error?: string;
};

// activate when installed or updated
browser.runtime.onInstalled.addListener(() => {
  console.log("Extension installed or updated");
});

// activate when browser starts
browser.runtime.onStartup.addListener(() => {
  console.log("Browser started");
});

browser.runtime.onMessage.addListener(async (message) => {
  if (message.type === "popup") {
    console.log("Popup opened");
  }
});

const port = browser.runtime.connectNative("com.pomdtr.webterm");
port.onMessage.addListener(async ({ id, payload }: Message) => {
  console.log("Received message", payload);
  switch (payload.command) {
    case "tab.list": {
      const tabs = await browser.tabs.query({});
      port.postMessage({ id, payload: tabs } as Message);
      return;
    }
    case "tab.focus": {
      const { tabId } = payload;
      const tab = await browser.tabs.update(tabId, { active: true });
      if (tab.windowId !== undefined) {
        await browser.windows.update(tab.windowId, { focused: true });
      }
      port.postMessage({ id, payload: { tab } } as Message);
      return;
    }
    case "tab.remove": {
      const { tabId } = payload;
      await browser.tabs.remove(tabId);
      port.postMessage({ id, payload: {} } as Message);
      return;
    }
    case "tab.reload": {
      const { tabId } = payload;
      const tab = await browser.tabs.reload(tabId);
      port.postMessage({ id, payload: { tab } } as Message);
      return;
    }
    case "tab.update": {
      const { tabId, url } = payload;
      const tab = await browser.tabs.update(tabId, { url });
      port.postMessage({ id, payload: { tab } } as Message);
      return;
    }
    case "tab.create": {
      const { url } = payload;
      const tab = await browser.tabs.create({ url });
      port.postMessage({ id, payload: { tab } } as Message);
      return;
    }
    case "window.list": {
      const windows = await browser.windows.getAll({});
      port.postMessage({ id, payload: windows } as Message);
      return;
    }
    case "window.focus": {
      const { windowId } = payload;
      const window = await browser.windows.update(windowId, {
        focused: true,
      });
      port.postMessage({ id, payload: { window } } as Message);
      return;
    }
    case "window.remove": {
      const { windowId } = payload;
      await browser.windows.remove(windowId);
      port.postMessage({ id, payload: {} } as Message);
      return;
    }
    case "window.create": {
      const { url } = payload;
      const window = await browser.windows.create({ url });
      port.postMessage({ id, payload: { window } } as Message);
      return;
    }
    case "bookmark.list": {
      const bookmarks = await browser.bookmarks.getTree();
      port.postMessage({ id, payload: bookmarks } as Message);
      return;
    }
    case "bookmark.create": {
      const { parentId, title, url } = payload;
      const bookmark = await browser.bookmarks.create({
        parentId,
        title,
        url,
      });
      port.postMessage({ id, payload: bookmark } as Message);
      return;
    }
    case "bookmark.remove": {
      const { id } = payload;
      await browser.bookmarks.remove(id);
      port.postMessage({ id, payload: {} } as Message);
      return;
    }
    case "download.list": {
      const downloads = await browser.downloads.search({});
      port.postMessage({ id, payload: downloads } as Message);
      return;
    }
    case "history.search": {
      const history = await browser.history.search({ text: payload.query });
      port.postMessage({ id, payload: history } as Message);
      return;
    }
    default: {
      console.error("Unknown message type", payload.type);
      port.postMessage({
        id,
        error: "Unknown message type",
      } as Message);
    }
  }
});
