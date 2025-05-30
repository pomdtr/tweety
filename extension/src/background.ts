import browser from "webextension-polyfill";

browser.runtime.onInstalled.addListener(() => {
  console.log("Extension installed");
});

browser.action.onClicked.addListener(() => {
  browser.tabs.create({ url: browser.runtime.getURL("terminal.html") });
});

browser.runtime.onMessage.addListener((_msg, _, sendResponse) => {
  console.log("Message received from content script or popup");
  sendResponse({ response: "Message received" });
  return true;
})

const port = browser.runtime.connectNative("com.github.pomdtr.tweety");

port.onMessage.addListener((message) => {
  port.postMessage({ response: "Message received" });
})

port.onDisconnect.addListener(() => {
  console.log("Native messaging host disconnected");
});
