import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";

async function main() {
  const targetElem = document.getElementById("terminal");
  if (!targetElem) {
    console.error("Terminal element not found");
    return;
  }

  const themeLigt = JSON.parse(targetElem.getAttribute("data-theme-light") || "{}");
  const themeDark = JSON.parse(
    targetElem.getAttribute("data-theme-dark") || "{}",
  );

  const params = new URLSearchParams(globalThis.location.search);
  let origin: URL;
  if (params.has("port")) {
    origin = new URL(`http://localhost:${params.get("port")}`);
  } else {
    origin = new URL(globalThis.location.origin);
  }


  const terminal = new Terminal({
    cursorBlink: true,
    allowProposedApi: true,
    macOptionIsMeta: true,
    macOptionClickForcesSelection: true,
    fontSize: 13,
    fontFamily: "Consolas,Liberation Mono,Menlo,Courier,monospace",
    theme: globalThis.matchMedia("(prefers-color-scheme: dark)").matches
      ? themeDark
      : themeLigt,
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

  const resp = await fetch("/_tweety/exec", {
    method: "POST",
  });

  if (!resp.ok) {
    console.error("Failed to create terminal");
    return;
  }

  const terminalID = await resp.text();
  if (!resp.ok) {
    console.error("Failed to create terminal");
    return;
  }

  const websocketProtocol = origin.protocol === "https:" ? "wss" : "ws";
  const websocketUrl = new URL(
    `${websocketProtocol}://${origin.host}/_tweety/pty/${terminalID}`,
  );

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
    await fetch(new URL(`/_tweety/resize/${terminalID}`, origin), {
      method: "POST",
      body: JSON.stringify({ cols, rows }),
    });
  });

  const attachAddon = new AttachAddon(ws);
  terminal.loadAddon(attachAddon);

  globalThis
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", function (e) {
      terminal.options.theme = e.matches ? themeDark : themeLigt;
    });

  globalThis.onbeforeunload = () => {
    ws.close();
  };

  globalThis.onfocus = () => {
    terminal.focus();
  };

  terminal.focus();
}

main();
