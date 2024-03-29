#!/usr/bin/env -S deno run -A
import * as path from "https://deno.land/std@0.208.0/path/mod.ts";
import * as fs from "https://deno.land/std@0.203.0/fs/mod.ts";

const __dirname = new URL(".", import.meta.url).pathname;

const chromeDir = path.join(__dirname, "..", "extension", "chrome");
const firefoxDir = path.join(__dirname, "..", "extension", "firefox");

Deno.removeSync(firefoxDir, { recursive: true });
fs.copySync(chromeDir, firefoxDir);

const manifestPath = path.join(firefoxDir, "manifest.json");
const chromeManifest = JSON.parse(Deno.readTextFileSync(manifestPath));

const firefoxManifest = chromeManifest;

firefoxManifest.background = {
  scripts: ["worker.js"],
};

firefoxManifest.side_panel = undefined;
firefoxManifest.permissions = ["storage", "contextMenus"];

Deno.writeTextFileSync(manifestPath, JSON.stringify(firefoxManifest, null, 2));
