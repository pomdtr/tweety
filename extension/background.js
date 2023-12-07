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
        chrome.windows.create({ url: origin });
    }
}
