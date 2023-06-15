import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";

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
  // don't register protocol handler if we're in a popup
  const searchParams = new URLSearchParams(window.location.search);
  let command: string = searchParams.get("command") || "";
  let dir: string = searchParams.get("dir") || "";

  // wake up background script
  const tabUrl = await chrome.runtime.sendMessage({ type: "popup" });

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

  // check if wesh server is running
  let ready = false;
  while (!ready) {
    try {
      const res = await fetch("http://localhost:9999/ready");
      if (res.status !== 200) {
        throw new Error("not ready");
      }
      ready = true;
    } catch (e) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
  }

  let url = `ws://localhost:9999/pty?cols=${terminal.cols}&rows=${
    terminal.rows
  }&url=${encodeURIComponent(tabUrl)}`;

  if (command) {
    url += `&command=${encodeURIComponent(command)}`;
  }
  if (dir) {
    url += `&dir=${encodeURIComponent(dir)}`;
  }

  const ws = new WebSocket(url);

  ws.onclose = () => {
    window.close();
  };

  const attachAddon = new AttachAddon(ws);
  terminal.loadAddon(attachAddon);

  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", function (e) {
      console.log("color scheme changed", e.matches);
      terminal.options.theme = e.matches ? darkTheme : lightTheme;
    });

  window.onfocus = () => {
    terminal.focus();
  };

  terminal.focus();
}

main();
