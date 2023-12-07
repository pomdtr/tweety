import { ITheme, Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import { WebglAddon } from "xterm-addon-webgl";
import { WebLinksAddon } from "xterm-addon-web-links";
import { AttachAddon } from "xterm-addon-attach";
import { nanoid } from "nanoid";
import { Config } from "./config";

async function importTheme(name: string) {
    return fetchJSON(`${import.meta.env.BASE_URL}themes/${name}.json`) as Promise<ITheme>
}

async function fetchJSON(url: string | URL, options?: RequestInit) {
    const resp = await fetch(url, options)
    return resp.json()
}

async function main() {
    const params = new URLSearchParams(window.location.search);
    let origin: URL
    if (params.has("port")) {
        const portNumber = params.get("port")
        const csp = document.getElementById("CSP")
        csp?.setAttribute("content", `default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self' ws://localhost:${portNumber} http://localhost:${portNumber}`)
        origin = new URL(`http://localhost:${portNumber}`)
    } else if (__TWEETY_ORIGIN__) {
        origin = new URL(__TWEETY_ORIGIN__)
    } else {
        origin = new URL(window.location.origin)
    }

    const config = await fetchJSON(new URL("/config", origin)) as Config
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
    const websocketProtocol = origin.protocol === "https:" ? "wss" : "ws";
    const websocketUrl = new URL(`${websocketProtocol}://${origin.host}/pty/${terminalID}`)

    websocketUrl.searchParams.set("cols", terminal.cols.toString())
    websocketUrl.searchParams.set("rows", terminal.rows.toString())

    const profileID = params.get("profile") || config.defaultProfile;
    const profile = config.profiles[profileID];
    if (!profile) {
        terminal.writeln(`Profile not found: ${profileID}`);
        return;
    }

    document.title = [profile.command, ...profile.args || []].join(" ");

    if (profile.favicon) {
        const link = document.getElementById("favicon") as HTMLLinkElement | null;
        if (link) {
            link.href = profile.favicon;
        }
    }

    websocketUrl.searchParams.set("profile", profileID);
    const ws = new WebSocket(websocketUrl);
    ws.onclose = () => {
        if (params.has("reload")) {
            window.location.reload();
        } else {
            window.opener = window;
            window.close()
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
        ws.close();
    }

    window.onfocus = () => {
        terminal.focus();
    };

    terminal.focus();
}

main();
