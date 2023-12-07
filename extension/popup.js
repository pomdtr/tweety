const iframe = document.createElement("iframe");

const { origin = "http://localhost:9999" } = chrome.storage.local.get([
  "origin",
]);
iframe.src = origin;
document.body.appendChild(iframe);

window.addEventListener("message", (event) => {
  if (event.source !== iframe.contentWindow) {
    console.error("Message not from iframe");
    return;
  }
  if (event.data !== "close") {
    console.error("Message not close");
    return;
  }
  window.close();
});
