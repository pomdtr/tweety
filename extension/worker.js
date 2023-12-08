chrome.runtime.onInstalled.addListener(function () {
    console.log("onInstalled");
    chrome.contextMenus.removeAll();
    chrome.contextMenus.create({
        id: "create_terminal_window",
        title: "Create Terminal Window",
        contexts: ["action"],
    });
});

chrome.runtime.onStartup.addListener(async function () {
    try {
        console.log("Connecting to native messaging host");
        chrome.runtime.connectNative("com.pomdtr.tweety")
    } catch (e) {
        console.log("Failed to connect to native messaging host: " + e);
    }
})

chrome.contextMenus.onClicked.addListener(function (info) {
    handleCommand(info.menuItemId);
});

chrome.action.onClicked.addListener(function () {
    handleCommand("create_terminal_tab");
});

chrome.commands.onCommand.addListener(handleCommand);

async function handleCommand(command) {
    const { origin = "http://localhost:9999" } = await chrome.storage.local.get([
        "origin",
    ]);
    if (command == "create_terminal_tab") {
        chrome.tabs.create({ url: origin });
    } else if (command == "create_terminal_window") {
        const [width, height] = [800, 600];
        const win = await chrome.windows.getCurrent();
        if (!win) {
            chrome.windows.create({ url: origin, type: "popup", width, height });
        }
        const left = Math.round(win.left + (win.width - width) / 2);
        const top = Math.round(win.top + (win.height - height) / 2);
        chrome.windows.create({ url: origin, type: "popup", width, height, left, top });
    }
}

chrome.omnibox.onInputStarted.addListener(async () => {
    const { origin = "http://localhost:9999" } = await chrome.storage.local.get([
        "origin",
    ]);
    const resp = await fetch(new URL("/config", origin))
    const config = await resp.json();
    chrome.storage.session.set({ config })
})

chrome.omnibox.onInputChanged.addListener(async function (text, suggest) {
    const { config = {} } = await chrome.storage.session.get(["config"])
    const profiles = Object.keys(config.profiles || {}).filter(profile => profile.includes(text))

    suggest(
        profiles.map((profile) => ({
            content: profile,
            description: profile
        }))
    )
});

chrome.omnibox.onInputEntered.addListener(async function (text) {
    const { origin = "http://localhost:9999" } = await chrome.storage.local.get([
        "origin",
    ]);
    chrome.tabs.create({ url: origin + "?profile=" + encodeURIComponent(text) });
});
