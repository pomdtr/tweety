#!/usr/bin/env -S deno run -A

import { program } from 'npm:@commander-js/extra-typings'

async function runCommand(command: string, args?: string[]): Promise<number> {
    const cmd = new Deno.Command(command, { args });
    const process = cmd.spawn();
    const status = await process.status;
    return status.code
}

program.name("tweety").action(async () => {
    Deno.exitCode = await runCommand("fish")
})

program.command("htop").action(async () => {
    Deno.exitCode = await runCommand("htop");
})

program.command("ssh").argument("<host>").action(async (host: string) => {
    Deno.exitCode = await runCommand("ssh", [host]);
})

program.command("kak").option("-f, --file <file>", "File to open in Kakoune").action(async (options) => {
    Deno.exitCode = await runCommand("kak", options.file ? [options.file] : []);
})


if (import.meta.main) {
    await program.parseAsync();
}
