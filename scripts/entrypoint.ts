#!/usr/bin/env -S deno run -A

import { program } from 'npm:@commander-js/extra-typings'

async function runCommand(command: string, args?: string[]): Promise<number> {
    const cmd = new Deno.Command(command, { args });
    const process = cmd.spawn();
    const status = await process.status;
    return status.code
}

program.name("tweety").action(async () => {
    await runCommand("fish")
})

program.command("htop").action(async () => {
    await runCommand("htop");
})

program.command("ssh").argument("<host>").action(async (host: string) => {
    await runCommand("ssh", [host]);
})


if (import.meta.main) {
    await program.parseAsync();
}
