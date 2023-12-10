chrome.runtime.onInstalled.addListener(function () {
    console.log("onInstalled");
    chrome.contextMenus.removeAll();
    chrome.contextMenus.create({
        id: "create_terminal_window",
        title: "Create Terminal Window",
        contexts: ["action"],
    });
});

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

chrome.omnibox.onInputChanged.addListener(function (text) {
    chrome.omnibox.setDefaultSuggestion({
        description: `Open <match>${text}</match> profile`,
    });
});


chrome.omnibox.onInputEntered.addListener(async function (text) {
    const { origin = "http://localhost:9999" } = await chrome.storage.local.get([
        "origin",
    ]);
    chrome.tabs.create({ url: origin + "?profile=" + encodeURIComponent(text) });
});
