const { origin = "http://localhost:9999" } = await chrome.storage.local.get([
  "origin",
]);

const url = new URL(origin);
const params = new URLSearchParams(window.location.search);
for (const param of params) {
  url.searchParams.set(param[0], param[1]);
}

const iframe = document.createElement("iframe");
iframe.src = url.toString();
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
