# Tweety - An Integrated Terminal for your Browser

Minimize your context switching by interacting with your terminal directly from your browser.

![tweety running from the browser](./static/tabs.png)

## Features

## Installation

Tweety is available on macOS, Linux.

```sh
# Homebrew (recommended)
brew install pomdtr/tap/tweety
```

or download a binary from [releases](https://github.com/pomdtr/tweety/releases).

If you want to compile it yourself, you can use the following command:

```sh
git clone https://github.com/pomdtr/tweety
cd tweety
make install
```

## Usage

```sh
tweety <command>
```

By default, tweety will start on port 9999, so you can access it at <http://localhost:9999>.

You can pass arguments to your entrypoint script using the `args` query parameter. The provided command will be splitted by the [shlex](https://pkg.go.dev/github.com/google/shlex) library, then passed as arguments to your entrypoint script.

- `http://localhost:9999/?args=ssh+example.com` will run the command `<entrypoint> ssh example.com`
- `http://localhost:9999/?args=nvim+/home/pomdtr/.zshrc` will run the command `<entrypoint> nvim /home/pomdtr/.zshrc`

Make sure to properly parse and validate params in your entrypoint script.

## Example entrypoint

```ts
#!/usr/bin/env -S deno run --allow-run

import { program } from 'npm:@commander-js/extra-typings'
import { existsSync } from "jsr:@std/fs"

// little helper to run commands
async function run(command: string, ...args: string[]) {
    const cmd = new Deno.Command(command, { args });
    const process = cmd.spawn();
    await process.status;
}

// handle http://localhost:9999/
program.action(async () => {
    await run("bash")
})

// handle http://localhost:9999?args=htop
program.command("htop").action(async () => {
    await run("htop");
})

// handle http://localhost:9999?args=ssh+<host>
program.command("ssh").argument("<host>").action(async (host: string) => {
    await run("ssh", host);
})

// handle http://localhost:9999?args=config
program.command("config").action(async () => {
    const scriptPath = new URL(import.meta.url).pathname;
    await run("nvim", scriptPath)
})

// handle http://localhost:9999?args=nvim+<file>
program.command("nvim").argument("<file>").action(async (file) => {
    // protect use again `nvim 'term://<malicious-command>'`
    if (file.startsWith("term://")) {
        console.error("Invalid file path: cannot use 'term://' prefix");
        Deno.exitCode = 1;
        return;
    }

    await run("nvim", file)
})

if (import.meta.main) {
    // parse arguments and run the appropriate command
    await program.parseAsync();
}
```
