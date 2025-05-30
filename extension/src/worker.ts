import browser from "webextension-polyfill";

browser.runtime.onInstalled.addListener(() => {
  console.log("Extension installed");
});

browser.action.onClicked.addListener(() => {
  browser.tabs.create({ url: browser.runtime.getURL("tty.html") });
});

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

  if (!("method" in msg) || typeof msg.method !== "string") {
    console.error("Received message without 'method' property:", msg);
    return false
  }

  if (!("params" in msg) || typeof msg.params !== "object" || msg.params === null) {
    console.error("Received message without 'params' property:", msg);
    return false
  }

  switch (msg.method) {
    case "getConfig": {
      console.log("Getting config from storage");
      browser.storage.session.get("nativePort").then((result) => {
        sendResponse({
          port: result.nativePort || null,
        })
      })
      break;
    }
    case "exec": {
      nativePort.postMessage(msg)
      break
    }
    case "resize": {
      nativePort.postMessage(msg)
      break;
    }
  }

  return false; // Keep the message channel open for sendResponse
})


const nativePort = browser.runtime.connectNative("com.github.pomdtr.tweety");

nativePort.onMessage.addListener(async (message) => {
  if (typeof message !== "object" || message === null) {
    console.error("Received non-object message from native messaging host:", message);
    return;
  }

  console.log("Message received from native messaging host:", message);
  await browser.runtime.sendMessage(message)
})

nativePort.onDisconnect.addListener(() => {
  console.log("Native messaging host disconnected");
});
