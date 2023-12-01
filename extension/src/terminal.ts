import { ITheme, Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";
import { nanoid } from "nanoid";
import { Config } from "./config";

const themeModules = import.meta.glob("./themes/*.json")
function importTheme(name: string) {
  const module = themeModules[`./themes/${name}.json`]
  if (!module) {
    throw new Error(`Theme ${name} not found`)
  }
  return module() as Promise<ITheme>
}

async function main() {
  const { port, token, config } =
    await chrome.storage.session.get(["port", "token", "config"]) as { port: number, token: string, config: Config }
  console.log(port, token, config)
  const lightTheme = await importTheme(config.theme || "Tomorrow")
  const darkTheme = await importTheme(config.themeDark || config.theme || "Tomorrow Night")
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

  // check if popcorn server is running
  let ready = false;
  while (!ready) {
    try {
      const res = await fetch(`http://localhost:${port}/ready`);
      if (res.status !== 200) {
        throw new Error("not ready");
      }
      ready = true;
    } catch (e) {
      await new Promise((resolve) => setTimeout(resolve, 1000));
    }
  }

  const terminalID = nanoid();
  const websocketUrl = new URL(`ws://localhost:${port}/pty/${terminalID}`)
  websocketUrl.searchParams.set("token", token)
  websocketUrl.searchParams.set("cols", terminal.cols.toString())
  websocketUrl.searchParams.set("rows", terminal.rows.toString())

  const params = new URLSearchParams(window.location.search);
  const profile = params.get("profile") || config.defaultProfile;
  websocketUrl.searchParams.set("profile", profile);

  const ws = new WebSocket(websocketUrl);
  ws.onclose = () => {
    window.close();
  };

  window.onresize = () => {
    fitAddon.fit();
  };

  terminal.onResize((size) => {
    const { cols, rows } = size;
    const url = `http://localhost:${port}/resize/${terminalID}?cols=${cols}&rows=${rows}`;
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
