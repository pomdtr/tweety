import browser from "webextension-polyfill";

type Message = {
  id: string;
  payload: any;
  error?: string;
};

async function getActiveTabId() {
  const activeTabs = await browser.tabs.query({
    active: true,
    currentWindow: true,
  });
  const tabId = activeTabs[0].id;
  if (tabId === undefined) {
    throw new Error("Active tab not found");
  }
  return tabId;
}

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
    case "tab.get": {
      let { tabId } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }
      const tab = await browser.tabs.get(tabId);
      port.postMessage({ id, payload: tab } as Message);
      return;
    }
    case "tab.pin": {
      let { tabId } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }
      const tab = await browser.tabs.update(tabId, { pinned: true });
      port.postMessage({ id, payload: tab } as Message);
      return;
    }
    case "tab.unpin": {
      let { tabId } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }
      const tab = await browser.tabs.update(tabId, { pinned: false });
      port.postMessage({ id, payload: tab } as Message);
      return;
    }
    case "tab.focus": {
      const { tabId } = payload;
      const tab = await browser.tabs.update(tabId, { active: true });
      if (tab.windowId !== undefined) {
        await browser.windows.update(tab.windowId, { focused: true });
      }
      port.postMessage({ id } as Message);
      return;
    }
    case "tab.remove": {
      let { tabIds } = payload;
      if (tabIds === undefined) {
        const activeTabs = await browser.tabs.query({
          active: true,
          currentWindow: true,
        });
        tabIds = activeTabs;
      }
      await browser.tabs.remove(tabIds);
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
      await browser.windows.update(tab.windowId!, {
        focused: true,
      });
      port.postMessage({ id, payload: { tab } } as Message);
      return;
    }
    case "tab.source": {
      let { tabId } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }

      const res = await chrome.scripting.executeScript({
        target: { tabId },
        func: () => {
          return document.documentElement.outerHTML;
        },
      });

      const html = res[0].result;

      port.postMessage({ id, payload: html } as Message);
      return;
    }
    case "selection.get": {
      let { tabId } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }

      const res = await chrome.scripting.executeScript({
        target: { tabId },
        func: () => {
          return window.getSelection()?.toString() || "";
        },
      });

      port.postMessage({ id, payload: res[0].result } as Message);
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
    case "extension.list": {
      const extensions = await browser.management.getAll();
      port.postMessage({ id, payload: extensions } as Message);
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
