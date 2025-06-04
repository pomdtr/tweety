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

    const xtermResp = await chrome.runtime.sendMessage<RequestGetXtermConfig, ResponseGetXtermConfig>({
        jsonrpc: "2.0",
        id: crypto.randomUUID(),
        method: "xterm.getConfig",
        params: {
            variant: window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light"
        }
    })

    if ("error" in xtermResp) {
        console.error("Error getting Xterm config:", xtermResp.error);
        return;
    }

    const requestId = crypto.randomUUID();
    const searchParams = new URLSearchParams(window.location.search);
    let params: RequestCreateTTY["params"]
    if (searchParams.has("app")) {
        params = {
            app: searchParams.get("app")!,
            args: searchParams.getAll("arg"),
        }
    }

    const resp = await chrome.runtime.sendMessage<RequestCreateTTY, ResponseCreateTTY>({
        jsonrpc: "2.0",
        id: requestId,
        method: "tty.create",
        params
    })

    if ("error" in resp) {
        console.error("Error creating TTY:", resp.error);
        globalThis.document.body.innerHTML = `<h1>Error: ${resp.error.message}</h1>`;
        return;
    }

    const terminal = new Terminal(xtermResp.result);
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

    window.matchMedia("(prefers-color-scheme: dark)").addEventListener("change", async (event) => {
        const variant = event.matches ? "dark" : "light";
        const resp = await chrome.runtime.sendMessage<RequestGetXtermConfig, ResponseGetXtermConfig>({
            jsonrpc: "2.0",
            id: crypto.randomUUID(),
            method: "xterm.getConfig",
            params: { variant }
        });
        if ("error" in resp) {
            console.error("Error getting Xterm config:", resp.error);
            return;
        }


        terminal.options.theme = resp.result.theme
    });

    terminal.focus();
}

main();
