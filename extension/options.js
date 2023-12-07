const { origin = "http://localhost:9999" } = await chrome.storage.local.get(["origin"]);

const textInput = document.getElementById("origin");
textInput.value = origin;

textInput.addEventListener("input", async () => {
    try {
        const url = new URL(textInput.value);
        textInput.setAttribute("aria-invalid", "false");
        await chrome.storage.local.set({ origin: url.toString() });
    } catch (_) {
        textInput.setAttribute("aria-invalid", "true")
    }
});
