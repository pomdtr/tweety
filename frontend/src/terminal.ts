import { ITheme, Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";
import { nanoid } from "nanoid";
import { Config } from "./config";

async function fetchTheme(name: string, origin?: string | URL) {
  const themeUrl = new URL(
    `/themes/${name}.json`,
    origin || window.location.origin
  );
  return fetchJSON(themeUrl) as Promise<ITheme>;
}

async function fetchJSON(url: string | URL, options?: RequestInit) {
  const resp = await fetch(url, options);
  return resp.json();
}

async function fetchText(url: string | URL, options?: RequestInit) {
  const resp = await fetch(url, options);
  return resp.text();
}

type QueryParams = {
  key: string | null
  command: string | null
  port: number | null
  profile: string | null
  reload: boolean
}

function parseQueryParams(search: string): QueryParams {
  const params = new URLSearchParams(search)
  const key = params.get("key")
  const port = params.get("port")
  return {
    command: params.get("command"),
    key: params.get("key"),
    port: port ?  parseInt(port) : null,
    profile: params.get("profile") || params.get("p") || null,
    reload: params.has("reload")
  }
}



async function main() {
  const params = parseQueryParams(window.location.search)
  let origin: URL;
  if (params.port) {
    origin = new URL(`http://localhost:${params.port}`);
  } else if (__TWEETY_ORIGIN__) {
    origin = new URL(__TWEETY_ORIGIN__);
  } else {
    origin = new URL(window.location.origin);
  }

  // check if tweety is running
  try {
    const resp = await fetch(new URL("/ping", origin));
    if (!resp.ok) {
      throw new Error("Not running");
    }
  } catch (e) {
    // if tweety is not running, show the readme
    const terminal = new Terminal({
      cursorBlink: true,
      allowProposedApi: true,
      macOptionIsMeta: true,
      macOptionClickForcesSelection: true,
      fontSize: 13,
      fontFamily: "Consolas,Liberation Mono,Menlo,Courier,monospace",
      theme: window.matchMedia("(prefers-color-scheme: dark)").matches
        ? await fetchTheme("Tomorrow Night", window.location.origin)
        : await fetchTheme("Tomorrow", window.location.origin),
    });

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.loadAddon(new WebglAddon());
    terminal.loadAddon(new WebLinksAddon());

    terminal.open(document.getElementById("terminal")!);

    fitAddon.fit();
    window.onresize = () => {
      fitAddon.fit();
    };

    const readme = window.matchMedia("(prefers-color-scheme: dark)").matches
      ? await fetchText("/readme/dark.ansi")
      : await fetchText("/readme/light.ansi");
    const rows = readme.split("\n");
    for (const row of rows) {
      terminal.writeln(row);
    }
    terminal.scrollToTop();
    return;
  }

  const config = (await fetchJSON(new URL("/config", origin))) as Config;
  const lightTheme = await fetchTheme(config.theme || "Tomorrow", origin);
  const darkTheme = await fetchTheme(
    config.themeDark || config.theme || "Tomorrow Night",
    origin
  );
  const terminal = new Terminal({
    cursorBlink: true,
    allowProposedApi: true,
    macOptionIsMeta: true,
    macOptionClickForcesSelection: true,
    fontSize: 13,
    fontFamily: "Consolas,Liberation Mono,Menlo,Courier,monospace",
    theme: window.matchMedia("(prefers-color-scheme: dark)").matches
      ? darkTheme
      : lightTheme,
    ...config.xterm,
  });

  const fitAddon = new FitAddon();
  const webLinksAddon = new WebLinksAddon(
    (event, uri) => {
      // check if cmd key is pressed
      if (event.metaKey || event.ctrlKey) {
        window.open(uri, "_blank");
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
    }
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
  const terminalID = nanoid();
  const websocketProtocol = origin.protocol === "https:" ? "wss" : "ws";
  const websocketUrl = new URL(
    `${websocketProtocol}://${origin.host}/pty/${terminalID}`
  );

  websocketUrl.searchParams.set("cols", terminal.cols.toString());
  websocketUrl.searchParams.set("rows", terminal.rows.toString());

  const profileID =
    params.profile || config.defaultProfile;
  const profile = config.profiles[profileID];
  if (!profile) {
    terminal.writeln(`Profile not found: ${profileID}`);
    return;
  }

  document.title = [profile.command, ...(profile.args || [])].join(" ");

  if (profile.favicon) {
    const link = document.getElementById("favicon") as HTMLLinkElement | null;
    if (link) {
      link.href = profile.favicon;
    }
  }

  websocketUrl.searchParams.set("profile", profileID);
  const ws = new WebSocket(websocketUrl);
  ws.onclose = () => {
    if (params.reload) {
      window.location.reload();
    } else {
      console.log("sending close message");
      window.parent.postMessage("close", "*");
      window.opener = window;
      window.close();
    }
  };

  window.onresize = () => {
    fitAddon.fit();
  };

  terminal.onTitleChange((title) => {
    document.title = title;
  });

  terminal.onResize((size) => {
    const { cols, rows } = size;
    const url = new URL(`/resize/${terminalID}`, origin);
    fetch(url, {
      method: "POST",
      body: JSON.stringify({ cols, rows }),
    });
  });

  const attachAddon = new AttachAddon(ws);
  terminal.loadAddon(attachAddon);

  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", function (e) {
      terminal.options.theme = e.matches ? darkTheme : lightTheme;
    });

  window.onbeforeunload = () => {
    ws.onclose = () => {};
    ws.close();
  };

  window.onfocus = () => {
    terminal.focus();
  };

  terminal.focus();
}

main();
