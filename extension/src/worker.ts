import browser from "webextension-polyfill";

browser.action.onClicked.addListener(() => {
  browser.tabs.create({ url: browser.runtime.getURL("tty.html") });
});

const nativePort = browser.runtime.connectNative("com.github.pomdtr.tweety");

nativePort.onMessage.addListener(async (message) => {
  if (typeof message !== "object" || message === null) {
    console.error("Received non-object message from native messaging host:", message);
    return;
  }

  // Check if the message is a valid JSON-RPC request
  if (!("method" in message) || typeof message.method !== "string") {
    return;
  }

  console.log("Message received from native messaging host:", message);


  if (message.method === "get_tabs") {
    if (!("id" in message) || typeof message.id !== "string") {
      console.error("Received get_tabs request without id:", message);
      return;
    }

    const tabs = await browser.tabs.query({});
    return nativePort.postMessage({
      jsonrpc: "2.0",
      id: message.id,
      result: {
        tabs
      }
    });
  } else if (message.method === "create_tab") {
    if (!("id" in message) || typeof message.id !== "string") {
      console.error("Received create_tab request without id:", message);
      return;
    }

    if (!("params" in message) || typeof message.params !== "object" || message.params === null) {
      console.error("Received create_tab request without params:", message);
      return;
    }

    if (!("url" in message.params) || typeof message.params.url !== "string") {
      console.error("Received create_tab request without url:", message);
      return;
    }

    const tab = await browser.tabs.create({ url: message.params.url });
    return nativePort.postMessage({
      jsonrpc: "2.0",
      id: message.id,
      result: {
        tab
      }
    });
  }

  await browser.runtime.sendMessage(message)
})

// @ts-ignore
browser.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (sender.id !== browser.runtime.id) {
    console.warn("Received message from unknown sender:", sender.id);
    return false; // Ignore messages from unknown senders
  }

  console.log("Message received from content script or popup", msg);

  if (typeof msg !== "object" || msg === null) {
    console.error("Received non-object message:", msg);
    return false
  }

  // if the jsonrpc request does not have an id, it is a notification
  if (!("id" in msg)) {
    nativePort.postMessage(msg)
    return false
  }

  const listener = (res: unknown) => {
    if (typeof res !== "object" || res === null) {
      return;
    }

    if (!("id" in res) || typeof res.id !== "string" || res.id !== msg.id) {
      return;
    }

    nativePort.onMessage.removeListener(listener);
    sendResponse(res);
  }

  nativePort.onMessage.addListener(listener)
  nativePort.postMessage(msg);
  return true
})
