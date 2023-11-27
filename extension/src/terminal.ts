import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";
import { nanoid } from "nanoid";

const imports = import.meta.glob("./themes/*.json")
const themes: Record<string, any> = {}
for (const [key, value] of Object.entries(imports)) {
  const name = key.slice("./themes/".length, -".json".length)
  themes[name] = await value() as any
}

async function main() {
  const lightTheme = themes["Tomorrow"];
  const darkTheme = themes["Tomorrow Night"];
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
  });

  const webglAddon = new WebglAddon();
  const fitAddon = new FitAddon();
  const webLinksAddon = new WebLinksAddon();
  terminal.loadAddon(fitAddon);
  terminal.loadAddon(webglAddon);
  terminal.loadAddon(webLinksAddon);

  terminal.open(document.getElementById("terminal")!);
  fitAddon.fit();

  const { port: popcornPort, token: popcornToken } =
    await chrome.storage.session.get(["port", "token"]);

  // check if popcorn server is running
  let ready = false;
  while (!ready) {
    try {
      const res = await fetch(`http://localhost:${popcornPort}/ready`);
      if (res.status !== 200) {
        throw new Error("not ready");
      }
      ready = true;
    } catch (e) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
  }

  const terminalID = nanoid();
  const websocketUrl = new URL(`ws://localhost:${popcornPort}/pty/${terminalID}`)
  websocketUrl.searchParams.set("token", popcornToken)
  websocketUrl.searchParams.set("cols", terminal.cols.toString())
  websocketUrl.searchParams.set("rows", terminal.rows.toString())
  if (window.location.hash === "#popup") {
    websocketUrl.searchParams.set("popup", "1")
  }

  const params = new URLSearchParams(window.location.search);
  const profile = params.get("profile");
  if (profile) {
    websocketUrl.searchParams.set("profile", profile);
  }

  const ws = new WebSocket(websocketUrl);
  ws.onclose = () => {
    window.close();
  };

  window.onresize = () => {
    fitAddon.fit();
  };

  terminal.onResize((size) => {
    const { cols, rows } = size;
    const url = `http://localhost:${popcornPort}/${terminalID}/resize?cols=${cols}&rows=${rows}`;
    fetch(url, {
      method: "POST",
    });
  });

  const attachAddon = new AttachAddon(ws);
  terminal.loadAddon(attachAddon);

  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", function (e) {
      terminal.options.theme = e.matches ? darkTheme : lightTheme;
    });

  window.onfocus = () => {
    terminal.focus();
  };

  terminal.focus();
}

main();
