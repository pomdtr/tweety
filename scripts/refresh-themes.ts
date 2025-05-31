#!/usr/bin/env deno run -A

import degit from "npm:degit";
import * as path from "https://deno.land/std@0.208.0/path/mod.ts";
import { existsSync } from "https://deno.land/std@0.203.0/fs/mod.ts";

const emitter = await degit("mbadolato/iTerm2-Color-Schemes/vscode", {
  force: true,
  verbose: true,
});

emitter.on("info", (info: { message: string }) => {
  console.log(info.message);
});

const cloneDir = path.join(Deno.makeTempDirSync(), "degit");
await emitter.clone(cloneDir);

const entries = await Deno.readDirSync(cloneDir);
const keyMapping = {
  "terminal.foreground": "foreground",
  "terminal.background": "background",
  "terminal.ansiBlack": "ansiBlack",
  "terminal.ansiBlue": "ansiBlue",
  "terminal.ansiCyan": "ansiCyan",
  "terminal.ansiGreen": "ansiGreen",
  "terminal.ansiMagenta": "ansiMagenta",
  "terminal.ansiRed": "ansiRed",
  "terminal.ansiWhite": "ansiWhite",
  "terminal.ansiYellow": "ansiYellow",
  "terminal.ansiBrightBlack": "ansiBrightBlack",
  "terminal.ansiBrightBlue": "ansiBrightBlue",
  "terminal.ansiBrightCyan": "ansiBrightCyan",
  "terminal.ansiBrightGreen": "ansiBrightGreen",
  "terminal.ansiBrightMagenta": "ansiBrightMagenta",
  "terminal.ansiBrightRed": "ansiBrightRed",
  "terminal.ansiBrightWhite": "ansiBrightWhite",
  "terminal.ansiBrightYellow": "ansiBrightYellow",
  "terminal.selectionBackground": "selectionBackground",
  "terminalCursor.foreground": "cursor",
} as Record<string, string>;

const themeDir = path.join(import.meta.dirname!, "..", "embed", "themes");
if (existsSync(themeDir)) {
  Deno.removeSync(themeDir, { recursive: true });
}
Deno.mkdirSync(themeDir);

for (const entry of entries) {
  const vscodeTheme = JSON.parse(
    Deno.readTextFileSync(path.join(cloneDir, entry.name))
  );
  const xtermTheme: Record<string, string> = {};
  for (const [key, value] of Object.entries(
    vscodeTheme["workbench.colorCustomizations"]
  )) {
    xtermTheme[keyMapping[key]] = value as string;
  }
  await Deno.writeTextFileSync(
    path.join(themeDir, entry.name),
    JSON.stringify(xtermTheme, null, 4)
  );
}

Deno.removeSync(path.join(cloneDir), { recursive: true });
