# Tweety

An integrated terminal for your browser.

![Tweety](./media/screenshot.jpg)

## Usage

```
tweety [-H <host>] [-p <port>]
```

## Configuration

Use the `~/.config/tweety/tweety.json` file to configure Tweety.

```json
{
  "theme": "Tomorrow",
  "themeDark": "Tomorrow Night",
  "env": {
    "EDITOR": "kak"
  },
  "defaultProfile": "default",
  "profiles": {
    "default": {
      "shell": "bash",
      "args": ["--login"],
      "env": {
        "EDITOR": "vim"
      }
    },
    "fish": {
      "shell": "fish",
      "args": ["--login"]
    }
  }
}
```

## Endpoints

- `/` - Open Default Profile
- `/p/<profile>` - Open Profile
- `/config` - View Configuration
