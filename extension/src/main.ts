import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebglAddon } from "@xterm/addon-webgl";
import { WebLinksAddon } from "@xterm/addon-web-links";
import { AttachAddon } from "@xterm/addon-attach";
import { split } from "shlex";

async function getTheme(variant: "light" | "dark") {
    const { theme } = await chrome.runtime.sendMessage({
        method: "getTheme",
        params: {
            variant,
        },
    });

    return theme;
}

async function main() {
    const targetElem = document.getElementById("terminal");
    if (!targetElem) {
        console.error("Terminal element not found");
        return;
    }

    const terminal = new Terminal({
        cursorBlink: true,
        allowProposedApi: true,
        macOptionIsMeta: true,
        macOptionClickForcesSelection: true,
        fontSize: 13,
        fontFamily: "Consolas,Liberation Mono,Menlo,Courier,monospace",
        theme: globalThis.matchMedia("(prefers-color-scheme: dark)").matches
            ? await getTheme("dark")
            : await getTheme("light"),
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
    const args = url.searchParams.get("args");
    const { id } = await chrome.runtime.sendMessage({
        method: "exec",
        params: {
            args: args ? split(args) : [],
            cwd: url.searchParams.get("cwd"),
        }
    })

    const websocketUrl = new URL(`ws://localhost:9999/_tweety/pty/${id}`,);

    websocketUrl.searchParams.set("cols", terminal.cols.toString());
    websocketUrl.searchParams.set("rows", terminal.rows.toString());

    const ws = new WebSocket(websocketUrl);

    globalThis.onresize = () => {
        fitAddon.fit();
    };

    terminal.onTitleChange((title) => {
        document.title = title;
    });

    terminal.onResize(async (size) => {
        const { cols, rows } = size;
        await chrome.runtime.sendMessage({
            method: "resize",
            params: {
                id,
                cols,
                rows,
            },
        })
    });

    const attachAddon = new AttachAddon(ws);
    terminal.loadAddon(attachAddon);

    globalThis
        .matchMedia("(prefers-color-scheme: dark)")
        .addEventListener("change", async function (e) {
            terminal.options.theme = e.matches ? await getTheme("dark") : await getTheme("light");
        });

    globalThis.onbeforeunload = () => {
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

    globalThis.onfocus = () => {
        terminal.focus();
    };

    terminal.focus();
}

main();
