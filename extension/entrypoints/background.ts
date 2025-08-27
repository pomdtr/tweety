import { JSONRPCRequest, JSONRPCResponse } from "~/entrypoints/shared/rpc"
import { Base64 } from 'js-base64';
import { type Browser } from 'wxt/browser';

export default defineBackground(() => {
  let _nativePort: Browser.runtime.Port | null = null;

  async function getNativePort(): Promise<Browser.runtime.Port | null> {
    if (_nativePort) {
      return _nativePort;
    }

    const port = browser.runtime.connectNative("com.github.pomdtr.tweety");

    const connected = await new Promise<boolean>((resolve) => {
      const onDisconnect = () => {
        port.onDisconnect.removeListener(onDisconnect);

        if (browser.runtime.lastError) {
          console.warn("Failed to connect:", browser.runtime.lastError.message);
          resolve(false);
        } else {
          console.warn("Disconnected from native messaging host");
          resolve(false);
        }
      };

      port.onDisconnect.addListener(onDisconnect);

      // Give the port a small window to disconnect if the host doesn't exist.
      setTimeout(() => {
        // Still connected after timeout — assume success
        resolve(true);
      }, 100); // 100ms is enough — disconnect happens almost instantly on failure
    });

    if (!connected) {
      return null;
    }

    _nativePort = port;

    _nativePort.onDisconnect.addListener(() => {
      _nativePort = null;
      if (browser.runtime.lastError) {
        console.warn("Port disconnected:", browser.runtime.lastError.message);
      } else {
        console.warn("Native messaging host disconnected");
      }
    });

    registerHandlers(_nativePort);

    let { browserId } = await browser.storage.local.get<{ browserId?: string }>("browserId");
    if (!browserId) {
      browserId = generateSecureId(12);
      await browser.storage.local.set({ browserId });
    }

    await initialize(_nativePort, browserId);

    return _nativePort;
  }

  function initialize(port: Browser.runtime.Port, browserId: string) {
    return new Promise((resolve) => {
      const requestId = crypto.randomUUID();

      port.onMessage.addListener((message) => {
        if (!isJsonRpcResponse(message) || message.id !== requestId) {
          return;
        }

        return resolve(message);
      });

      port.postMessage({
        jsonrpc: "2.0",
        method: "initialize",
        id: requestId,
        params: {
          browserId,
          version: browser.runtime.getManifest().version,
        }
      })
    })
  }

  browser.runtime.onInstalled.addListener(async () => {
    browser.sidePanel?.setPanelBehavior({
      openPanelOnActionClick: false
    });

    await getNativePort();
  });


  // should not be async, else side panel will not open when invoked from the keyboard shortcut
  async function handleCommand(commandId: string, input?: unknown) {
    if (commandId === 'openInNewTab') {
      await browser.tabs.create({
        url: browser.runtime.getURL("/term.html"),
        active: true,
      });
    } else if (commandId === 'openInNewWindow') {
      await browser.windows.create({
        url: browser.runtime.getURL("/term.html"),
        focused: true,
      });
    } else if (commandId.startsWith("commands:")) {
      const [_, command] = commandId.split(":");
      console.log("Running command:", command);
      const nativePort = await getNativePort()
      if (!nativePort) {
        console.warn("Native host is not connected");
        return;
      }

      const msg: JSONRPCRequest = {
        jsonrpc: "2.0",
        id: crypto.randomUUID(),
        method: "commands.run",
        params: {
          command,
          input
        },
      }

      const listener = (res: unknown) => {
        if (typeof res !== "object" || res === null) {
          return;
        }

        if (!isJsonRpcResponse(res)) {
          console.error("Received invalid JSON-RPC response:", res);
          return;
        }

        if (res.id !== msg.id) {
          return;
        }

        if (res.error) {
          browser.notifications.create({
            type: "basic",
            iconUrl: browser.runtime.getURL("/icon/128.png"),
            title: "Command Error",
            message: res.error.message,
          });
        }

        nativePort.onMessage.removeListener(listener);
      }

      nativePort.onMessage.addListener(listener)
      nativePort.postMessage(msg);
    }
  }

  browser.contextMenus.onClicked.addListener(async (info) => {
    if (typeof info.menuItemId !== 'string') {
      console.warn("Invalid menuItemId:", info.menuItemId);
      return;
    }

    await handleCommand(info.menuItemId, {
      linkUrl: info.linkUrl,
      srcUrl: info.srcUrl,
      pageUrl: info.pageUrl,
      frameUrl: info.frameUrl,
      selectionText: info.selectionText,
      mediaType: info.mediaType,
    });
  })

  browser.commands.onCommand.addListener(async (command) => {
    await handleCommand(command);
  });

  function setContextMenus(commands: { id: string, meta: { title: string, contexts: string[], documentUrlPatterns?: string[], targetUrlPatterns?: string[] } }[]) {
    browser.contextMenus.removeAll();

    browser.contextMenus.create({
      id: 'openInNewTab',
      title: 'Open in New Tab',
      contexts: ['all'],
    });

    browser.contextMenus.create({
      id: 'openInNewWindow',
      title: 'Open in New Window',
      contexts: ['all'],
    });

    if (commands.length === 0) {
      return;
    }

    browser.contextMenus.create({
      id: "runCommand",
      title: "Run Command",
      contexts: ['all'],
    });

    for (const command of commands) {
      console.log("Registering context menu for command:", command);
      browser.contextMenus.create({
        id: `commands:${command.id}`,
        parentId: "runCommand",
        title: command.meta.title,
        // @ts-ignore
        contexts: command.meta.contexts,
        documentUrlPatterns: command.meta.documentUrlPatterns,
        targetUrlPatterns: command.meta.targetUrlPatterns,
      });
    }
  }

  function registerHandlers(nativePort: Browser.runtime.Port) {
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
          case "fetch": {
            try {
              const resp = await fetch(params[0], params[1])
              sendResponse({
                status: resp.status,
                // @ts-ignore
                headers: Object.fromEntries(resp.headers.entries()),
                body: await Base64.fromUint8Array(new Uint8Array(await resp.arrayBuffer())),
              })
            } catch (error) {
              console.error("Fetch error:", error);
              sendError({ code: -32000, message: `Fetch failed: ${(error as Error).message}` });
            }
            break;
          }
          case "commands.update": {
            console.log("Updating commands:", params[0]);
            setContextMenus(params[0])
            break;
          }
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
          case "tabs.print":
            try {
              let targetTabId: number;

              if (params.length === 0) {
                const currentTab = await browser.tabs.query({ active: true, lastFocusedWindow: true });
                if (currentTab.length === 0 || !currentTab[0].id) {
                  sendError({ code: -32602, message: "No active tab found" });
                  return;
                }
                targetTabId = currentTab[0].id;
              } else {
                targetTabId = params[0];
              }

              const results = await browser.scripting.executeScript({
                target: { tabId: targetTabId },
                func: () => document.documentElement.outerHTML,
              });
              console.log("Tab content retrieved for printing:", results[0].result);
              sendResponse(results[0].result);
            } catch (error) {
              sendError({ code: -32000, message: `Failed to get tab content: ${(error as Error).message}` });
            }
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

  }

  browser.runtime.onMessage.addListener((msg, sender, sendResponse) => {
    if (sender.id !== browser.runtime.id) {
      console.warn("Received message from unknown sender:", sender.id);
      return false; // Ignore messages from unknown senders
    }

    getNativePort().then((nativePort) => {
      if (!nativePort) {
        return sendResponse({
          jsonrpc: "2.0",
          id: msg.id || generateSecureId(12),
          error: {
            code: -32001,
            message: "Native host is not connected",
          }
        });
      }

      const listener = (res: unknown) => {
        if (typeof res !== "object" || res === null) {
          return;
        }

        if (!isJsonRpcResponse(res)) {
          return;
        }

        if (res.id !== msg.id) {
          return;
        }

        nativePort.onMessage.removeListener(listener);
        sendResponse(res);
      }

      nativePort.onMessage.addListener(listener)
      nativePort.postMessage(msg);
    })


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
});
