import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";
import { nanoid } from "nanoid";

const darkTheme = {
  foreground: "#c5c8c6",
  background: "#1d1f21",
  black: "#000000",
  blue: "#81a2be",
  cyan: "#8abeb7",
  green: "#b5bd68",
  magenta: "#b294bb",
  red: "#cc6666",
  white: "#ffffff",
  yellow: "#f0c674",
  brightBlack: "#000000",
  brightBlue: "#81a2be",
  brightCyan: "#8abeb7",
  brightGreen: "#b5bd68",
  brightMagenta: "#b294bb",
  brightRed: "#cc6666",
  brightWhite: "#ffffff",
  brightYellow: "#f0c674",
  selectionBackground: "#373b41",
  cursor: "#c5c8c6",
};

const lightTheme = {
  foreground: "#4d4d4c",
  background: "#ffffff",
  black: "#000000",
  blue: "#4271ae",
  cyan: "#3e999f",
  green: "#718c00",
  magenta: "#8959a8",
  red: "#c82829",
  white: "#ffffff",
  yellow: "#eab700",
  brightBlack: "#000000",
  brightBlue: "#4271ae",
  brightCyan: "#3e999f",
  brightGreen: "#718c00",
  brightMagenta: "#8959a8",
  brightRed: "#c82829",
  brightWhite: "#ffffff",
  brightYellow: "#eab700",
  selectionBackground: "#d6d6d6",
  cursor: "#4d4d4c",
};

async function main() {
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

  const params = new URLSearchParams(window.location.search);

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
  let websocketUrl = `ws://localhost:${popcornPort}/pty/${terminalID}?token=${[
    popcornToken,
  ]}&cols=${terminal.cols}&rows=${terminal.rows}`;

  const profile = params.get("profile");
  if (profile) {
    websocketUrl += `&profile=${encodeURIComponent(profile)})}`;
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
