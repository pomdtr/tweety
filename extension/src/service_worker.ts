import { JSONRPCRequest, JSONRPCResponse } from "./rpc";

chrome.runtime.onInstalled.addListener(() => {
  chrome.sidePanel.setPanelBehavior({
    openPanelOnActionClick: false
  });

  chrome.contextMenus.create({
    id: 'openInNewTab',
    title: 'Open in new tab',
    contexts: ['action'],
  });
  chrome.contextMenus.create({
    id: 'openSidePanel',
    title: 'Open in side panel',
    contexts: ['action'],
  });
  chrome.contextMenus.create({
    id: 'openInNewWindow',
    title: 'Open in new window',
    contexts: ['action'],
  });

  // Separator between action commands and default behavior group
  chrome.contextMenus.create({
    type: 'separator',
    id: 'actionSeparator',
    contexts: ['action'],
  });

  // Radio group for default action behavior
  chrome.contextMenus.create({
    id: 'defaultBehavior',
    title: 'Action button behavior',
    type: 'normal',
    contexts: ['action'],
  });
  chrome.contextMenus.create({
    id: 'defaultBehavior_newTab',
    parentId: 'defaultBehavior',
    title: 'Open in new tab',
    type: 'radio',
    contexts: ['action'],
    checked: true,
  });
  chrome.contextMenus.create({
    id: 'defaultBehavior_sidePanel',
    parentId: 'defaultBehavior',
    title: 'Open in side panel',
    type: 'radio',
    contexts: ['action'],
    checked: false,
  });
});

// Store and use the selected default behavior
chrome.contextMenus.onClicked.addListener((info) => {
  if (typeof info.menuItemId !== 'string') {
    return
  }

  if (!info.menuItemId.startsWith('defaultBehavior_')) {
    return; // Ignore clicks on other menu items
  }

  chrome.sidePanel.setPanelBehavior({
    openPanelOnActionClick: info.menuItemId === 'defaultBehavior_sidePanel',
  })
});

// Override the action button click to use the selected default behavior
chrome.action.onClicked.addListener(() => {
  chrome.tabs.create({
    url: chrome.runtime.getURL("tty.html"),
    active: true,
  });
})

// should not be async, else side panel will not open when invoked from the keyboard shortcut
function handleCommand(commandId: string) {
  if (commandId === 'openInNewTab') {
    chrome.tabs.create({
      url: chrome.runtime.getURL("tty.html"),
      active: true,
    });
  } else if (commandId === 'openInSidePanel') {
    chrome.tabs.query({ active: true, currentWindow: true }, ([tab]) => {
      chrome.sidePanel.open({ tabId: tab.id! });
    });

  } else if (commandId === 'openInNewWindow') {
    chrome.windows.create({
      url: chrome.runtime.getURL("tty.html"),
      focused: true,
    });
  }
}

chrome.contextMenus.onClicked.addListener((info) => {
  if (typeof info.menuItemId !== 'string') {
    console.warn("Invalid menuItemId:", info.menuItemId);
    return;
  }


  handleCommand(info.menuItemId);
})

chrome.commands.onCommand.addListener(async (command) => {
  handleCommand(command);
});

const nativePort = chrome.runtime.connectNative("com.github.pomdtr.tweety");

chrome.storage.local.get<{ browserId?: string; }>("browserId", async ({ browserId }) => {
  if (!browserId) {
    browserId = generateSecureId(12);
    await chrome.storage.local.set({ browserId });
  }

  nativePort.postMessage({
    jsonrpc: "2.0",
    method: "initialize",
    params: {
      browserId,
      version: chrome.runtime.getManifest().version,
    }
  })
})


nativePort.onMessage.addListener(async (message) => {
  if (!isJsonRpcRequest(message)) {
    return;
  }

  const { id, method, params } = message;

  // Helper to send JSON-RPC response
  const sendResponse = (result: unknown) => nativePort.postMessage({
    jsonrpc: "2.0",
    id,
    result
  });

  // Helper to send JSON-RPC error
  const sendError = (error: unknown) => nativePort.postMessage({
    jsonrpc: "2.0",
    id,
    error
  });
  if (!Array.isArray(params)) {
    console.error("Invalid params: expected an array", params);
    sendError({ code: -32602, message: "Invalid params: expected an array" });
    return
  }

  console.log("Received message:", message);
  try {
    switch (method) {
      // Tabs methods
      case "tabs.query":
        const tabs = await chrome.tabs.query(params[0]);
        sendResponse(tabs);
        break;
      case "tabs.get":
        if (params.length == 0) {
          const currentTab = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
          if (currentTab.length === 0 || !currentTab[0].id) {
            sendError({ code: -32602, message: "No active tab found" });
            return;
          }

          const tab = await chrome.tabs.get(currentTab[0].id);
          sendResponse(tab);
          break
        }
        const tab = await chrome.tabs.get(params[0]);
        sendResponse(tab);
        break;
      case "tabs.create":
        const newTab = await chrome.tabs.create(params[0]);
        sendResponse(newTab);
        break;
      case "tabs.duplicate":
        const duplicatedTab = await chrome.tabs.duplicate(params[0]);
        sendResponse(duplicatedTab);
        break;
      case "tabs.discard":
        await chrome.tabs.discard(params[0]);
        sendResponse(null);
        break;
      case "tabs.remove":
        await chrome.tabs.remove(params[0]);
        sendResponse(null);
        break;
      case "tabs.captureVisibleTab":
        const capturedTab = await chrome.tabs.captureVisibleTab();
        sendResponse(capturedTab);
        break;
      case "tabs.update":
        const resp = await chrome.tabs.update(params[0], params[1]);
        sendResponse(resp);
        break;
      case "tabs.reload":
        await chrome.tabs.reload(params[0], params[1]);
        sendResponse(null);
        break;
      case "tabs.goForward":
        await chrome.tabs.goForward(params[0]);
        sendResponse(null);
        break;
      case "tabs.goBack":
        await chrome.tabs.goBack(params[0]);
        sendResponse(null);
        break;
      case "windows.getAll":
        const windows = await chrome.windows.getAll();
        sendResponse(windows);
        break;
      case "windows.get":
        const window = await chrome.windows.get(params[0]);
        sendResponse(window);
        break;
      case "windows.getCurrent":
        const currentWindow = await chrome.windows.getCurrent();
        sendResponse(currentWindow);
        break;
      case "windows.getLastFocused":
        const lastFocusedWindow = await chrome.windows.getLastFocused();
        sendResponse(lastFocusedWindow);
        break;
      case "windows.create":
        const newWindow = await chrome.windows.create(params[0]);
        sendResponse(newWindow);
        break;
      case "windows.remove":
        await chrome.windows.remove(params[0]);
        sendResponse(null);
        break;
      case "history.search":
        const historyItems = await chrome.history.search(params[0]);
        sendResponse(historyItems);
        break;
      case "bookmarks.getTree":
        const bookmarksTree = await chrome.bookmarks.getTree();
        sendResponse(bookmarksTree);
        break;
      case "bookmarks.getRecent":
        const recentBookmarks = await chrome.bookmarks.getRecent(params[0]);
        sendResponse(recentBookmarks);
        break;
      case "bookmarks.search":
        const searchResults = await chrome.bookmarks.search(params[0]);
        sendResponse(searchResults);
        break;
      case "bookmarks.create":
        const createdBookmark = await chrome.bookmarks.create(params[0]);
        sendResponse(createdBookmark);
        break;
      case "bookmarks.update":
        const updatedBookmark = await chrome.bookmarks.update(params[0], params[1]);
        sendResponse(updatedBookmark);
        break;
      case "bookmarks.remove":
        await chrome.bookmarks.remove(params[0]);
        sendResponse(null);
        break;
      case "notifications.create":
        if (params.length == 2) {
          const res = await chrome.notifications.create(params[0], params[1]);
          await sendResponse(res);
          break;
        }

        if (params.length == 1) {
          const res = await chrome.notifications.create(params[0]);
          await sendResponse(res);
          break;
        }
        console.error("Invalid params for notifications.create:", params);
        sendError({ code: -32602, message: "Invalid params for notifications.create" });
        break;
      default:
        console.error("Method not found:", method);
        sendError({ code: -32601, message: `Method not found: ${method}` });
        break;
    }
  } catch (err) {
    console.error("Error handling message:", err);
    sendError({ code: -32000, message: (err as Error).message });
  }
});

chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (sender.id !== chrome.runtime.id) {
    console.warn("Received message from unknown sender:", sender.id);
    return false; // Ignore messages from unknown senders
  }

  console.log("Message received from content script or popup", msg);
  if (!isJsonRpcRequest(msg)) {
    return false; // Ignore invalid messages
  }

  const listener = (res: unknown) => {
    if (typeof res !== "object" || res === null) {
      return;
    }

    if (!isJsonRpcResponse(res)) {
      console.error("Received invalid JSON-RPC response:", res);
      return;
    }

    nativePort.onMessage.removeListener(listener);
    sendResponse(res);
  }

  nativePort.onMessage.addListener(listener)
  nativePort.postMessage(msg);
  return true
})

function isJsonRpcRequest(message: unknown): message is JSONRPCRequest {
  if (typeof message !== "object" || message === null) {
    return false;
  }

  if (!("jsonrpc" in message) || message.jsonrpc !== "2.0") {
    return false;
  }

  if (!("method" in message) || typeof message.method !== "string") {
    return false;
  }

  if ("id" in message && typeof message.id !== "string") {
    return false;
  }

  if ("params" in message && (typeof message.params !== "object" || message.params === null)) {
    return false;
  }

  return true;
}

const isJsonRpcResponse = (message: unknown): message is JSONRPCResponse => {
  if (typeof message !== "object" || message === null) {
    return false;
  }

  if (!("jsonrpc" in message) || message.jsonrpc !== "2.0") {
    return false;
  }

  if (!("id" in message) || typeof message.id !== "string") {
    return false;
  }

  if ("result" in message && (typeof message.result !== "object" || message.result === null)) {
    return false;
  }

  if ("error" in message && (typeof message.error !== "object" || message.error === null)) {
    return false;
  }

  return true;
}

function generateSecureId(length = 12) {
  const charset = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
  const array = new Uint8Array(length);
  crypto.getRandomValues(array);
  return Array.from(array, (byte) => charset[byte % charset.length]).join('');
}
