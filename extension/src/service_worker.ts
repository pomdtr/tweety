import { JSONRPCRequest, JSONRPCResponse } from "./rpc";
import browser from "webextension-polyfill";

browser.runtime.onInstalled.addListener(() => {
  if (chrome.sidePanel) {
    chrome.sidePanel.setPanelBehavior({
      openPanelOnActionClick: false
    });
  }
  // Set the default side panel path

  browser.contextMenus.create({
    id: 'openInNewTab',
    title: 'Open in new tab',
    contexts: ['action'],
  });
  browser.contextMenus.create({
    id: 'openInNewWindow',
    title: 'Open in new window',
    contexts: ['action'],
  });

  // Separator between action commands and default behavior group
  browser.contextMenus.create({
    type: 'separator',
    id: 'actionSeparator',
    contexts: ['action'],
  });

  // Radio group for default action behavior
  browser.contextMenus.create({
    id: 'defaultBehavior',
    title: 'Action button behavior',
    type: 'normal',
    contexts: ['action'],
  });
  browser.contextMenus.create({
    id: 'defaultBehavior_newTab',
    parentId: 'defaultBehavior',
    title: 'Open in new tab',
    type: 'radio',
    contexts: ['action'],
    checked: true,
  });
  browser.contextMenus.create({
    id: 'defaultBehavior_sidePanel',
    parentId: 'defaultBehavior',
    title: 'Open in side panel',
    type: 'radio',
    contexts: ['action'],
    checked: false,
  });
});

browser.runtime.onStartup.addListener(async () => {
  let { browserId } = browser.storage.local.get("browserId") as { browserId?: string };

  if (!browserId) {
    browserId = generateSecureId(12);
    await browser.storage.local.set({ browserId });
  }

  nativePort.postMessage({
    jsonrpc: "2.0",
    method: "initialize",
    params: {
      browserId,
      version: browser.runtime.getManifest().version,
    }
  })

})

// Store and use the selected default behavior
browser.contextMenus.onClicked.addListener((info) => {
  if (typeof info.menuItemId !== 'string') {
    return
  }

  if (!info.menuItemId.startsWith('defaultBehavior_')) {
    return; // Ignore clicks on other menu items
  }

  if (chrome.sidePanel) {
    chrome.sidePanel.setPanelBehavior({
      openPanelOnActionClick: info.menuItemId === 'defaultBehavior_sidePanel',
    })
  }
});

// Override the action button click to use the selected default behavior
browser.action.onClicked.addListener(() => {
  browser.tabs.create({
    url: browser.runtime.getURL("term.html"),
    active: true,
  });
})

// should not be async, else side panel will not open when invoked from the keyboard shortcut
async function handleCommand(commandId: string) {
  if (commandId === 'openInNewTab') {
    await browser.tabs.create({
      url: browser.runtime.getURL("term.html"),
      active: true,
    });
  } else if (commandId === 'openInNewWindow') {
    await browser.windows.create({
      url: browser.runtime.getURL("term.html"),
      focused: true,
    });
  }
}

browser.contextMenus.onClicked.addListener(async (info) => {
  if (typeof info.menuItemId !== 'string') {
    console.warn("Invalid menuItemId:", info.menuItemId);
    return;
  }

  await handleCommand(info.menuItemId);
})

browser.commands.onCommand.addListener(async (command) => {
  await handleCommand(command);
});

const nativePort = browser.runtime.connectNative("com.github.pomdtr.tweety");


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
        const tabs = await browser.tabs.query(params[0]);
        sendResponse(tabs);
        break;
      case "tabs.get":
        if (params.length == 0) {
          const currentTab = await browser.tabs.query({ active: true, lastFocusedWindow: true });
          if (currentTab.length === 0 || !currentTab[0].id) {
            sendError({ code: -32602, message: "No active tab found" });
            return;
          }

          const tab = await browser.tabs.get(currentTab[0].id);
          sendResponse(tab);
          break
        }
        const tab = await browser.tabs.get(params[0]);
        sendResponse(tab);
        break;
      case "tabs.create":
        const newTab = await browser.tabs.create(params[0]);
        sendResponse(newTab);
        break;
      case "tabs.duplicate":
        const duplicatedTab = await browser.tabs.duplicate(params[0]);
        sendResponse(duplicatedTab);
        break;
      case "tabs.discard":
        await browser.tabs.discard(params[0]);
        sendResponse(null);
        break;
      case "tabs.remove":
        await browser.tabs.remove(params[0]);
        sendResponse(null);
        break;
      case "tabs.captureVisibleTab":
        const capturedTab = await browser.tabs.captureVisibleTab();
        sendResponse(capturedTab);
        break;
      case "tabs.update":
        const resp = await browser.tabs.update(params[0], params[1]);
        sendResponse(resp);
        break;
      case "tabs.reload":
        await browser.tabs.reload(params[0], params[1]);
        sendResponse(null);
        break;
      case "tabs.goForward":
        await browser.tabs.goForward(params[0]);
        sendResponse(null);
        break;
      case "tabs.goBack":
        await browser.tabs.goBack(params[0]);
        sendResponse(null);
        break;
      case "windows.getAll":
        const windows = await browser.windows.getAll();
        sendResponse(windows);
        break;
      case "windows.get":
        const window = await browser.windows.get(params[0]);
        sendResponse(window);
        break;
      case "windows.getCurrent":
        const currentWindow = await browser.windows.getCurrent();
        sendResponse(currentWindow);
        break;
      case "windows.getLastFocused":
        const lastFocusedWindow = await browser.windows.getLastFocused();
        sendResponse(lastFocusedWindow);
        break;
      case "windows.create":
        const newWindow = await browser.windows.create(params[0]);
        sendResponse(newWindow);
        break;
      case "windows.remove":
        await browser.windows.remove(params[0]);
        sendResponse(null);
        break;
      case "windows.update":
        const updatedWindow = await browser.windows.update(params[0], params[1]);
        sendResponse(updatedWindow);
        break;
      case "history.search":
        const historyItems = await browser.history.search(params[0]);
        sendResponse(historyItems);
        break;
      case "bookmarks.getTree":
        const bookmarksTree = await browser.bookmarks.getTree();
        sendResponse(bookmarksTree);
        break;
      case "bookmarks.getRecent":
        const recentBookmarks = await browser.bookmarks.getRecent(params[0]);
        sendResponse(recentBookmarks);
        break;
      case "bookmarks.search":
        const searchResults = await browser.bookmarks.search(params[0]);
        sendResponse(searchResults);
        break;
      case "bookmarks.create":
        const createdBookmark = await browser.bookmarks.create(params[0]);
        sendResponse(createdBookmark);
        break;
      case "bookmarks.update":
        const updatedBookmark = await browser.bookmarks.update(params[0], params[1]);
        sendResponse(updatedBookmark);
        break;
      case "bookmarks.remove":
        await browser.bookmarks.remove(params[0]);
        sendResponse(null);
        break;
      case "notifications.create":
        if (params.length == 2) {
          const res = await browser.notifications.create(params[0], params[1]);
          await sendResponse(res);
          break;
        }

        if (params.length == 1) {
          const res = await browser.notifications.create(params[0]);
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

// @ts-ignore
browser.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (sender.id !== browser.runtime.id) {
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
