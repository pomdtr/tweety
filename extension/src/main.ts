import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { AttachAddon } from "@xterm/addon-attach";
import { WebglAddon } from "@xterm/addon-webgl";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { RequestCreateTTY, RequestGetXtermConfig, RequestResizeTTY, ResponseCreateTTY, ResponseGetXtermConfig } from "./rpc";

async function main() {
    const { result: config } = await chrome.runtime.sendMessage<RequestGetXtermConfig, ResponseGetXtermConfig>({
        jsonrpc: "2.0",
        id: crypto.randomUUID(),
        method: "config.get",
    })

    const terminal = new Terminal(config);
    const fitAddon = new FitAddon();

    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebglAddon());
    terminal.loadAddon(new WebLinksAddon());

    terminal.open(document.getElementById("terminal")!);
    fitAddon.fit();

    const requestId = crypto.randomUUID();

    const url = new URL(globalThis.location.href);
    const command = url.searchParams.get("command") || url.searchParams.get("cmd") || undefined;
    const resp = await chrome.runtime.sendMessage<RequestCreateTTY, ResponseCreateTTY>({
        jsonrpc: "2.0",
        id: requestId,
        method: "tty.create",
        params: {
            command,
            cols: terminal.cols,
            rows: terminal.rows,
        }
    })

    const ws = new WebSocket(resp.result.url);
    const attachAddon = new AttachAddon(ws);
    terminal.loadAddon(attachAddon);

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
