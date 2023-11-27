type Message = {
  id: string;
  payload?: {
    command: string;
    [key: string]: any;
  };
  error?: string;
};

enum ContextMenuID {
  OPEN_TERMINAL_TAB = "open-terminal-tab",
  COPY_EXTENSION_ID = "copy-extension-id",
}

// activate when installed or updated
chrome.runtime.onInstalled.addListener(() => {
  console.log("Extension installed or updated");
  chrome.contextMenus.create({
    id: ContextMenuID.OPEN_TERMINAL_TAB,
    title: "Open Terminal in New Tab",
    contexts: ["action"],
  });
  chrome.contextMenus.create({
    title: "Copy Extension ID",
    id: ContextMenuID.COPY_EXTENSION_ID,
    contexts: ["action"],
  });
});

// activate when chrome starts
chrome.runtime.onStartup.addListener(() => {
  console.log("Browser started");
});


const nativePort = chrome.runtime.connectNative("com.pomdtr.popcorn");
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

chrome.storage.session.setAccessLevel({
  accessLevel: "TRUSTED_AND_UNTRUSTED_CONTEXTS",
});

async function handleMessage(payload: any): Promise<any> {
  switch (payload.command) {
    case "init": {
      await chrome.storage.session.set({
        port: payload.port,
        token: payload.token,
      });
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
  switch (info.menuItemId) {
    case ContextMenuID.OPEN_TERMINAL_TAB: {
      await chrome.tabs.create({ url: mainPage });
      break;
    }
    case ContextMenuID.COPY_EXTENSION_ID: {
      await addToClipboard(chrome.runtime.id);
      break;
    }
    default: {
      throw new Error(`Unknown menu item: ${info.menuItemId}`);
    }
  }
});

chrome.commands.onCommand.addListener(async (command) => {
  switch (command) {
    case "open-terminal-tab": {
      const tab = await chrome.tabs.create({ url: "/src/terminal.html" });
      await chrome.windows.update(tab.windowId, { focused: true });
      break;
    }
    default: {
      throw new Error(`Unknown command: ${command}`);
    }
  }
})

chrome.omnibox.onInputStarted.addListener(async () => {
  chrome.omnibox.setDefaultSuggestion({
    description: "Run command",
  });
});

chrome.omnibox.onInputChanged.addListener(async (text) => {
  chrome.omnibox.setDefaultSuggestion({
    description: `Run: ${text}`,
  });
});

chrome.omnibox.onInputEntered.addListener(async (disposition) => {
  const url = `/src/terminal.html`;
  switch (disposition) {
    case "currentTab":
      await chrome.tabs.update({ url });
      break;
    case "newForegroundTab":
      await chrome.tabs.create({ url });
      break;
    case "newBackgroundTab":
      await chrome.tabs.create({ url, active: false });
  }
});

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
