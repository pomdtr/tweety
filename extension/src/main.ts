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
    const webLinksAddon = new WebLinksAddon(
        (event, uri) => {
            // check if cmd key is pressed
            if (event.metaKey || event.ctrlKey) {
                globalThis.open(uri, "_blank");
            }
        },
        {
            hover: (_, text) => {
                let tooltip = document.getElementById("tooltip");
                if (tooltip) {
                    if (tooltip.id === `${text}-tooltip`) {
                        return;
                    }

                    if (tooltip.matches(":hover")) {
                        return;
                    }
                    tooltip.remove();
                }

                const tooltipId = `${text}-tooltip`;
                tooltip = document.createElement("div");
                tooltip.id = tooltipId;
                tooltip.className = "tooltip";
                tooltip.style.position = "fixed";
                tooltip.innerHTML = `<a href="${text}" target="_blank">Follow link</a>`;
                tooltip.style.visibility = "hidden";
                document.body.appendChild(tooltip);

                const tootlipSize = tooltip.getBoundingClientRect();
                tooltip.style.zIndex = "1000";

                // Add a delay of 1 second before showing the tooltip
                setTimeout(() => {
                    // check if the tooltip still exists
                    const tooltip = document.getElementById(tooltipId);
                    if (tooltip) {
                        if (mouseX - tootlipSize.height > 0) {
                            tooltip.style.top = `${mouseY - tootlipSize.height}px`;
                        } else {
                            tooltip.style.top = `${mouseY}px`;
                        }

                        tooltip.style.left = `${mouseX - tootlipSize.width / 2}px`;
                        tooltip.style.visibility = "visible";
                    }
                }, 1000);
            },
            leave: (_, text) => {
                const tooltip = document.getElementById(`${text}-tooltip`);
                if (tooltip && !tooltip.matches(":hover")) {
                    tooltip.remove();
                    return;
                }

                tooltip!.addEventListener("mouseleave", () => {
                    tooltip!.remove();
                });
            },
        },
    );
    let [mouseX, mouseY] = [0, 0];
    document.addEventListener("mousemove", (e) => {
        mouseX = e.clientX;
        mouseY = e.clientY;
    });

    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebglAddon());
    terminal.loadAddon(webLinksAddon);

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
