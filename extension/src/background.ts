import { Config } from "./config";

type Message = {
  id: string;
  payload?: {
    command: string;
    [key: string]: any;
  };
  error?: string;
};

enum ContextMenuID {
  OPEN_PROFILE_DEFAUlT = "open-profile-default",
  OPEN_PROFILE = "open-profile",
  COPY_INSTALLATION_COMMAND = "copy-installation-command",
}

chrome.runtime.onInstalled.addListener(() => {
  console.log("Extension installed or updated");
  chrome.contextMenus.create({
    title: "Copy Installation Command",
    id: ContextMenuID.COPY_INSTALLATION_COMMAND,
    contexts: ["action"],
  });
});

let nativePort: chrome.runtime.Port;
try {
  nativePort = chrome.runtime.connectNative("com.pomdtr.popcorn");
  nativePort.onMessage.addListener(async (msg: Message) => {
    console.log("Received message", msg);
    try {
      const res = await handleMessage(msg.payload);
      nativePort.postMessage({
        id: msg.id,
        payload: res,
      });
    } catch (e: any) {
      nativePort.postMessage({
        id: msg.id,
        error: e.message,
      });
    }
  });
} catch (e) {
  console.log(`Native messaging host not found: ${e}`);
}

async function handleMessage(payload: any): Promise<any> {
  switch (payload.command) {
    case "init": {
      const { port, token, config } = payload as { port: number, token: string, config: Config }
      await chrome.storage.session.set({
        port, token, config
      });

      chrome.contextMenus.remove(ContextMenuID.COPY_INSTALLATION_COMMAND)
      chrome.contextMenus.create({
        id: ContextMenuID.OPEN_PROFILE_DEFAUlT,
        title: "Open Default Profile",
        contexts: ["action"],
      });

      chrome.contextMenus.create({
        id: ContextMenuID.OPEN_PROFILE,
        title: "Open Profile",
        contexts: ["action"],
      });
      for (const profile of Object.keys(config.profiles)) {
        chrome.contextMenus.create({
          id: `${ContextMenuID.OPEN_PROFILE}:${profile}`,
          parentId: ContextMenuID.OPEN_PROFILE,
          title: profile,
          contexts: ["action"],
        });
      }

      return "ok";
    }
    case "tab.list": {
      if (payload.allWindows) {
        return await chrome.tabs.query({});
      }

      if (payload.windowId !== undefined) {
        return await chrome.tabs.query({ windowId: payload.windowId });
      }

      return await chrome.tabs.query({ currentWindow: true });
    }
    case "tab.get": {
      let { tabId } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }
      return await chrome.tabs.get(tabId);
    }
    case "tab.pin": {
      let { tabIds } = payload;
      if (tabIds === undefined) {
        tabIds = [await getActiveTabId()];
      }

      for (const tabId of tabIds) {
        await chrome.tabs.update(tabId, { pinned: true });
      }

      return;
    }
    case "tab.unpin": {
      let { tabIds } = payload;
      if (tabIds === undefined) {
        tabIds = [await getActiveTabId()];
      }

      for (const tabId of tabIds) {
        await chrome.tabs.update(tabId, { pinned: false });
      }

      return;
    }
    case "tab.focus": {
      const { tabId } = payload;
      const tab = await chrome.tabs.update(tabId, { active: true });
      if (tab.windowId !== undefined) {
        await chrome.windows.update(tab.windowId, { focused: true });
      }
      return;
    }
    case "tab.remove": {
      let { tabIds } = payload;
      if (tabIds === undefined) {
        tabIds = [await getActiveTabId()];
      }
      await chrome.tabs.remove(tabIds);
      return;
    }
    case "tab.reload": {
      let { tabIds } = payload;
      if (tabIds === undefined) {
        tabIds = [await getActiveTabId()];
      }
      for (const tabId of tabIds) {
        await chrome.tabs.reload(tabId);
      }
      return;
    }
    case "tab.update": {
      let { tabId, url } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }
      await chrome.tabs.update(tabId, { url });
      return;
    }
    case "tab.create": {
      const { url, urls } = payload;
      const currentWindow = await chrome.windows.getCurrent();
      if (currentWindow.id === undefined) {
        throw new Error("Current window not found");
      }

      if (url !== undefined) {
        await chrome.tabs.create({ url, windowId: currentWindow.id });
        await chrome.windows.update(currentWindow.id, { focused: true });
        return;
      }

      for (const url of urls) {
        await chrome.tabs.create({ url, windowId: currentWindow.id });
      }

      await chrome.windows.update(currentWindow.id, { focused: true });
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

      return res[0].result;
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

      return res[0].result;
    }
    case "selection.set": {
      let { tabId, text } = payload;
      if (tabId === undefined) {
        tabId = await getActiveTabId();
      }

      console.log(`setting input to ${text}`);
      await chrome.scripting.executeScript({
        target: { tabId },
        args: [text],
        func: (text) => {
          // Get the current selection
          const selection = window.getSelection();
          if (!selection) {
            return;
          }

          if (selection.rangeCount > 0) {
            // Get the first range of the selection
            const range = selection.getRangeAt(0);

            // Create a new text node as replacement
            const newNode = document.createTextNode(text);

            // Replace the selected range with the new node
            range.deleteContents();
            range.insertNode(newNode);

            // Adjust the selection to the end of the inserted node
            range.collapse(false);

            // Clear any existing selection
            selection.removeAllRanges();

            // Add the modified range to the selection
            selection.addRange(range);
          }
        },
      });

      return;
    }
    case "window.list": {
      return chrome.windows.getAll({});
    }
    case "window.focus": {
      const { windowId } = payload;
      return await chrome.windows.update(windowId, {
        focused: true,
      });
    }
    case "window.remove": {
      const { windowId } = payload;
      await chrome.windows.remove(windowId);
      return;
    }
    case "window.create": {
      const { url } = payload;
      return await chrome.windows.create({ url });
    }
    case "extension.list": {
      return await chrome.management.getAll();
    }
    case "bookmark.list": {
      return await chrome.bookmarks.getTree();
    }
    case "bookmark.create": {
      const { parentId, title, url } = payload;
      return chrome.bookmarks.create({
        parentId,
        title,
        url,
      });
    }
    case "bookmark.remove": {
      const { id } = payload;
      chrome.bookmarks.remove(id);
      return;
    }
    case "download.list": {
      return await chrome.downloads.search({});
    }
    case "history.search": {
      return chrome.history.search({ text: payload.query });
    }
    default: {
      throw new Error(`Unknown command: ${payload.command}`);
    }
  }
}

async function getActiveTabId() {
  const activeTabs = await chrome.tabs.query({
    active: true,
    currentWindow: true,
  });
  const tabId = activeTabs[0].id;
  if (tabId === undefined) {
    throw new Error("Active tab not found");
  }
  return tabId;
}

chrome.contextMenus.onClicked.addListener(async (info) => {
  const mainPage = "/src/terminal.html";

  const menuItemID = info.menuItemId;
  if (typeof menuItemID !== "string") {
    throw new Error(`Unknown menu item: ${menuItemID}`);
  }

  if (menuItemID == ContextMenuID.OPEN_PROFILE_DEFAUlT) {
    await chrome.tabs.create({ url: mainPage });
  } else if (typeof menuItemID.startsWith(ContextMenuID.OPEN_PROFILE)) {
    const profile = menuItemID.split(":")[1];
    if (!profile) {
      throw new Error(`Unknown menu item: ${menuItemID}`);
    }
    await chrome.tabs.create({
      url: `${mainPage}?profile=${profile}`,
    });
  } else if (menuItemID == ContextMenuID.COPY_INSTALLATION_COMMAND) {
    throw new Error(`Unknown menu item: ${menuItemID}`);
  } else {
    await addToClipboard(`popcorn init ${chrome.runtime.id}`);
  }
});

chrome.action.onClicked.addListener(async () => {
  if (nativePort === undefined) {
    return;
  }
  await chrome.tabs.create({ url: "/src/terminal.html" });
});

// chrome.omnibox.onInputStarted.addListener(async () => {
//   chrome.omnibox.setDefaultSuggestion({
//     description: "Run command",
//   });
// });

// chrome.omnibox.onInputChanged.addListener(async (text) => {
//   chrome.omnibox.setDefaultSuggestion({
//     description: `Run: ${text}`,
//   });
// });

// chrome.omnibox.onInputEntered.addListener(async (disposition) => {
//   const url = `/src/terminal.html`;
//   switch (disposition) {
//     case "currentTab":
//       await chrome.tabs.update({ url });
//       break;
//     case "newForegroundTab":
//       await chrome.tabs.create({ url });
//       break;
//     case "newBackgroundTab":
//       await chrome.tabs.create({ url, active: false });
//   }
// });

async function addToClipboard(value: string) {
  await chrome.offscreen.createDocument({
    url: 'src/offscreen.html',
    reasons: [chrome.offscreen.Reason.CLIPBOARD],
    justification: 'Write text to the clipboard.'
  });

  // Now that we have an offscreen document, we can dispatch the
  // message.
  chrome.runtime.sendMessage({
    type: 'copy-data-to-clipboard',
    target: 'offscreen-doc',
    data: value
  });
}
