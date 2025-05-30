import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { AttachAddon } from "@xterm/addon-attach";
import { WebglAddon } from "@xterm/addon-webgl";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { split } from "shlex";

async function main() {
    // const config = await chrome.runtime.sendMessage({
    //     method: "getConfig",
    //     params: {}
    // })

    const terminal = new Terminal({
        cursorBlink: true,
        allowProposedApi: true,
        macOptionIsMeta: true,
        macOptionClickForcesSelection: true,
        fontSize: 13,
        fontFamily: "Consolas,Liberation Mono,Menlo,Courier,monospace",
        // theme: globalThis.matchMedia("(prefers-color-scheme: dark)").matches
        //     ? await getTheme("dark")
        //     : await getTheme("light"),
    });

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

    const url = new URL(globalThis.location.href);
    const argsParam = url.searchParams.get("args");

    const requestId = crypto.randomUUID();
    chrome.runtime.onMessage.addListener((message) => {
        console.log("Received message from extension worker:", message);

        if (message.id !== requestId) {
            return;
        }

        const ws = new WebSocket(message.result.url);
        const attachAddon = new AttachAddon(ws);
        terminal.loadAddon(attachAddon);

        terminal.onResize(async (size) => {
            const { cols, rows } = size;
            await chrome.runtime.sendMessage({
                jsonrpc: "2.0",
                method: "resize",
                params: {
                    id: message.result.id,
                    cols,
                    rows,
                },
            })
        });

        globalThis.onbeforeunload = () => {
            ws.onclose = () => { }
            ws.close();
        };

        ws.onclose = () => {
            terminal.writeln(
                "Connection closed. Hit Enter to refresh the page.",
            );

            terminal.onKey((event) => {
                if (event.key === "\r" || event.key === "\n") {
                    globalThis.location.reload();
                }
            });
        }
    })

    chrome.runtime.sendMessage({
        jsonrpc: "2.0",
        id: requestId,
        method: "exec",
        params: {
            args: argsParam ? split(argsParam) : [],
            cwd: url.searchParams.get("cwd"),
            cols: terminal.cols,
            rows: terminal.rows,
        }
    })

    globalThis.onresize = () => {
        fitAddon.fit();
    };

    terminal.onTitleChange((title) => {
        document.title = title;
    });

    globalThis.onfocus = () => {
        terminal.focus();
    };

    terminal.focus();

    // globalThis
    //     .matchMedia("(prefers-color-scheme: dark)")
    //     .addEventListener("change", async function (e) {
    //         terminal.options.theme = e.matches ? await getTheme("dark") : await getTheme("light");
    //     });
}

main();
