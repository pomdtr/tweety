import { JSONRPCRequest, JSONRPCResponse } from "./rpc";

chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: 'openInNewTab',
    title: 'Open in new tab',
    contexts: ['action'],
  })
  chrome.contextMenus.create({
    id: 'openInNewWindow',
    title: 'Open in new window',
    contexts: ['action'],
  });
  chrome.contextMenus.create({
    id: 'openinPopupWindow',
    title: 'Open in popup window',
    contexts: ['action'],
  })
})

async function handleCommand(commandId: string) {
  if (commandId === 'openInNewTab') {
    await chrome.tabs.create({
      url: chrome.runtime.getURL("tty.html"),
      active: true,
    });
  } else if (commandId === 'openInNewWindow') {
    chrome.windows.create({
      url: chrome.runtime.getURL("tty.html"),
      focused: true,
    });
  } else if (commandId === 'openinPopupWindow') {
    // Get the current window to calculate the center position
    const currentWindow = await chrome.windows.getCurrent();
    const screenWidth = currentWindow.width ?? 1200;
    const screenHeight = currentWindow.height ?? 800;
    const screenLeft = currentWindow.left ?? 0;
    const screenTop = currentWindow.top ?? 0;

    const popupWidth = 800;
    const popupHeight = 600;
    const left = Math.round(screenLeft + (screenWidth - popupWidth) / 2);
    const top = Math.round(screenTop + (screenHeight - popupHeight) / 2);

    chrome.windows.create({
      url: chrome.runtime.getURL("tty.html"),
      type: "popup",
      height: popupHeight,
      width: popupWidth,
      left,
      top,
      focused: true,
    });
  }
}

chrome.contextMenus.onClicked.addListener(async (info) => {
  if (typeof info.menuItemId !== 'string') {
    console.warn("Invalid menuItemId:", info.menuItemId);
    return;
  }

  await handleCommand(info.menuItemId);
})

chrome.commands.onCommand.addListener(async (command) => {
  await handleCommand(command);
});


chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (!tab) {
    console.warn("No active tab found for context menu action.");
    return;
  }

  if (info.menuItemId === 'openSidePanel') {
    // This will open the panel in all the pages on the current window.
    chrome.sidePanel.open({ windowId: tab.windowId });
  }
});

const nativePort = chrome.runtime.connectNative("com.github.pomdtr.tweety");

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
