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
tweety <entrypoint>
```

By default, tweety will start on port 9999, so you can access it at <http://localhost:9999>.

You can use the `--host` and `--port` flags to change the host and port:

```sh
tweety --host 0.0.0.0 --port 8080 <script-path>
```

You can pass arguments to your entrypoint script using the `cmd` query parameter. The provided command will be splitted by the [shlex](https://pkg.go.dev/github.com/google/shlex) library, then passed as arguments to your entrypoint script.

- `http://localhost:9999/?cmd=ssh+example.com` will run the command `<entrypoint> ssh example.com`
- `http://localhost:9999/?cmd=nvim+/home/pomdtr/.zshrc` will run the command `<entrypoint> nvim /home/pomdtr/.zshrc`

Make sure to properly parse and validate params in your entrypoint script.

## Example entrypoint

```ts
#!/usr/bin/env -S deno run -A

import { program } from 'npm:@commander-js/extra-typings'

async function runCommand(command: string, args?: string[]): Promise<number> {
    const cmd = new Deno.Command(command, { args });
    const process = cmd.spawn();
    const status = await process.status;
    return status.code
}

// url: http://localhost:9999/
program.name("tweety").action(async () => {
    Deno.exitCode = await runCommand("fish")
})

// url: http://localhost:9999/htop
program.command("htop").action(async () => {
    Deno.exitCode = await runCommand("htop");
})

// url: http://localhost:9999/ssh/example.com
program.command("ssh").argument("<host>").action(async (host: string) => {
    Deno.exitCode = await runCommand("ssh", [host]);
})

// url: http://localhost:9999/nvim?file=/path/to/file
program.command("edit").option("-f, --file <file>", "File to open in editor").action(async (options) => {
    Deno.exitCode = await runCommand("nvim", options.file ? [options.file] : []);
})

if (import.meta.main) {
    await program.parseAsync();
}
```
