import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { AttachAddon } from "@xterm/addon-attach";
import { WebglAddon } from "@xterm/addon-webgl";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { RequestCreateTTY, RequestGetXtermConfig, RequestResizeTTY, ResponseCreateTTY, ResponseGetXtermConfig } from "./rpc";

async function main() {
    const anchor = document.getElementById("terminal");
    if (!anchor) {
        console.error("terminal element not found");
        return;
    }

    const { result: config } = await chrome.runtime.sendMessage<RequestGetXtermConfig, ResponseGetXtermConfig>({
        jsonrpc: "2.0",
        id: crypto.randomUUID(),
        method: "xterm.getConfig",
    })

    const requestId = crypto.randomUUID();
    const urlParams = new URLSearchParams(window.location.search);
    const mode = urlParams.get("mode")
    let params: RequestCreateTTY["params"];
    if (mode == "editor") {
        if (!urlParams.has("file")) {
            console.error("File parameter is required for editor mode");
            return;
        }

        params = {
            mode: "editor",
            file: urlParams.get("file")!
        }
    } else if (mode == "ssh") {
        if (!urlParams.has("host")) {
            console.error("Host parameter is required for SSH mode");
            return;
        }
        params = {
            mode: "ssh",
            host: urlParams.get("host")!
        }
    } else {
        params = {
            mode: "terminal",
        }
    }

    const resp = await chrome.runtime.sendMessage<RequestCreateTTY, ResponseCreateTTY>({
        jsonrpc: "2.0",
        id: requestId,
        method: "tty.create",
        params: params
    })

    const terminal = new Terminal(config);
    const fitAddon = new FitAddon();

    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebglAddon());
    terminal.loadAddon(new WebLinksAddon());

    const ws = new WebSocket(resp.result.url);
    const attachAddon = new AttachAddon(ws);
    terminal.loadAddon(attachAddon);

    terminal.open(anchor);
    terminal.onResize(async (size) => {
        const { cols, rows } = size;
        await chrome.runtime.sendMessage<RequestResizeTTY>({
            jsonrpc: "2.0",
            method: "tty.resize",
            params: {
                tty: resp.result.id,
                cols,
                rows,
            },
        })
    });
    fitAddon.fit();

    globalThis.onbeforeunload = () => {
        ws.onclose = () => { }
        ws.close();
    };

    ws.onclose = async () => {
        globalThis.close();
    }

    globalThis.onresize = () => {
        fitAddon.fit();
    };

    terminal.onTitleChange((title) => {
        document.title = `${title}  |  Tweety`
    });

    globalThis.onfocus = () => {
        terminal.focus();
    };

    terminal.focus();
}

main();
