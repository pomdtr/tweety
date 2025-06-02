import browser from "webextension-polyfill";
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

    const { result: config } = await browser.runtime.sendMessage<RequestGetXtermConfig, ResponseGetXtermConfig>({
        jsonrpc: "2.0",
        id: crypto.randomUUID(),
        method: "xterm.getConfig",
    })

    const requestId = crypto.randomUUID();
    const searchParams = new URLSearchParams(window.location.search);
    const mode = searchParams.get("mode")
    let params: RequestCreateTTY["params"];
    if (mode == "editor") {
        if (!searchParams.has("file")) {
            console.error("File parameter is required for editor mode");
            return;
        }

        params = {
            mode: "editor",
            file: searchParams.get("file")!
        }
    } else if (mode == "ssh") {
        if (!searchParams.has("host")) {
            console.error("Host parameter is required for SSH mode");
            return;
        }
        params = {
            mode: "ssh",
            host: searchParams.get("host")!
        }
    } else if (mode == "app") {
        if (!searchParams.has("app")) {
            console.error("App parameter is required for app mode");
            return;
        }
        params = {
            mode: "app",
            app: searchParams.get("app")!
        }


    } else {
        params = {
            mode: "terminal",
        }
    }

    const resp = await browser.runtime.sendMessage<RequestCreateTTY, ResponseCreateTTY>({
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
        await browser.runtime.sendMessage<RequestResizeTTY>({
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
        if (searchParams.has("reload")) {
            globalThis.location.reload();
            return;
        }

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
