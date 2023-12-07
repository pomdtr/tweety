chrome.runtime.onInstalled.addListener(function () {
    console.log("Installed");
    chrome.contextMenus.create({
        id: "create_terminal_tab",
        title: "Create Terminal Tab",
        contexts: ["action"],
    });

    chrome.contextMenus.create({
        id: "create_terminal_window",
        title: "Create Terminal Window",
        contexts: ["action"],
    });
});

chrome.commands.onCommand.addListener(handleCommand);
chrome.contextMenus.onClicked.addListener(function (info) {
    handleCommand(info.menuItemId);
});

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
