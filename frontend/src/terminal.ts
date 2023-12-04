import { ITheme, Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";
import { nanoid } from "nanoid";
import { Config } from "./config";

async function importTheme(name: string) {
    return fetchJSON(`/themes/${name}.json`) as Promise<ITheme>
}

async function fetchJSON(url: string, options?: RequestInit) {
    const resp = await fetch(url, options)
    return resp.json()
}

async function main() {
    const config = await fetchJSON("/config") as Config
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
    const terminalID = nanoid();
    const websocketProtocol = window.location.protocol === "https:" ? "wss" : "ws";
    const websocketUrl = new URL(`${websocketProtocol}://${window.location.host}/pty/${terminalID}`)

    websocketUrl.searchParams.set("cols", terminal.cols.toString())
    websocketUrl.searchParams.set("rows", terminal.rows.toString())

    const params = new URLSearchParams(window.location.search);
    const profileID = params.get("profile") || config.defaultProfile;

    const profile = config.profiles[profileID];
    if (!profile) {
        terminal.writeln(`Profile not found: ${profileID}`);
        return;
    }

    if (profile.title) {
        document.title = profile.title;
    }

    if (profile.favicon) {
        const link = document.getElementById("favicon") as HTMLLinkElement | null;
        if (link) {
            link.href = profile.favicon;
        }
    }

    websocketUrl.searchParams.set("profile", profileID);
    const ws = new WebSocket(websocketUrl);
    ws.onclose = () => {
        window.opener = window;
        window.close()
    };

    window.onresize = () => {
        fitAddon.fit();
    };

    terminal.onTitleChange((title) => {
        if (profile.title) {
            title = `${title} - ${profile.title}`;
        }
        document.title = title;
    });


    terminal.onResize((size) => {
        const { cols, rows } = size;
        const url = `/resize/${terminalID}`;
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
        ws.close();
    }

    window.onfocus = () => {
        terminal.focus();
    };

    terminal.focus();
}

main();