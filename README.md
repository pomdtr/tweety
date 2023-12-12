# Tweety - An Integrated Terminal for your Browser

Minimize your context switching by interacting with your terminal directly from your browser.

## Features

### Mix Web and Terminal Tabs

![tweety running from the browser](./static/tabs.png)

### Side by Side Mode

![tweety running in a split pane](./static/panel.png)

### Developer-Console Integration

![tweety running in the developer panel](./static/devtools.png)

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

See the `tweety completion` command to generate completion scripts for your shell.



## Usage

```sh
tweety [-p <port>]
```

By default, tweety will start on port 9999, so you can access it at <http://localhost:9999>.

If you prefer to use a browser friendly URL, you can go to `https://tweety.sh` instead.
The page will access Tweety on port 9999.

> Note: Browser Friendly urls are not available on Safari, and requires to disable the Shield feature on Brave.

## Browser Extension

Browser extensions for Chrome and Firefox are available in the `extension` folder.

## Chrome Extension

- Clone the repository
- Go to `chrome://extensions`
- Enable `Developer mode`
- Click on `Load unpacked`
- Select the `extension/chrome` folder

By default, the extension will try to connect to `localhost:9999`. You can
customize the origin in the extension options.

## Firefox Extension

- Clone the repository
- Go to `about:debugging
- Click on `This Firefox`
- Click on `Load Temporary Add-on...`
- Select the `extension/firefox/manifest.json` file

## Starting Tweety on Boot

If you have installed Tweety using Homebrew, you can use the following command:

```sh
brew services start tweety
```

To unload the service, use:

```sh
brew services stop tweety
```

## Configuration

Use the `$XDG_CONFIG_DIR/tweety/tweety.json` file to configure Tweety (defaults
to `~/.config/tweety/tweety.json`).

Alternatively, you can use the `TWEETY_CONFIG` environment variable to specify a
custom path.

```json
{
  "$schema": "https://github.com/pomdtr/tweety/releases/latest/download/config.schema.json",
  "theme": "Tomorrow",
  "themeDark": "Tomorrow Night",
  "env": {
    "EDITOR": "kak"
  },
  "defaultProfile": "default",
  "profiles": {
    "default": {
      "command": "bash",
      "args": ["--login"],
      "env": {
        "EDITOR": "vim"
      }
    },
    "fish": {
      "command": "fish",
      "args": ["--login"],
      "favicon": "https://fishshell.com/favicon.ico"
    }
  }
}
```

## Endpoints

- `/` - open the default profile
- `/?reload=true` - reload the page when the command exits
- `/?profile=<profile>` - open a specific profile

## FAQ

### Windows Version?

The library used to create the terminal UI (xterm.js) does not support Windows yet.
But it's planned, so stay tuned!
