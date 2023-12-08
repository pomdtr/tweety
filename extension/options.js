const { origin = "http://localhost:9999", nativeMessaging } = await chrome.storage.local.get(["origin", "nativeMessaging"]);

const textInput = document.getElementById("origin");
textInput.value = origin;

const nativeMessagingCheckbox = document.getElementById("nativeMessaging");
nativeMessagingCheckbox.checked = nativeMessaging;

nativeMessagingCheckbox.addEventListener("change", async (event) => {
    await chrome.storage.local.set({ nativeMessaging: event.target.checked });
});


textInput.addEventListener("input", async (event) => {
    try {
        const url = new URL(event.target.value);
        textInput.setAttribute("aria-invalid", "false");
        await chrome.storage.local.set({ origin: url.toString() });
    } catch (_) {
        textInput.setAttribute("aria-invalid", "true")
    }
});
