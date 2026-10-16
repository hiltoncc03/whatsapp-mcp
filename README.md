# WhatsApp MCP Server — Linux/KDE Fork

**🌐 Language / Idioma:** [Português](./README.pt-BR.md) · **English**

> Fork of [`lharries/whatsapp-mcp`](https://github.com/lharries/whatsapp-mcp) maintained by [@hiltoncc03](https://github.com/hiltoncc03), focused on running reliably on Linux (tested on **CachyOS + KDE Plasma**).

A **Model Context Protocol (MCP)** server for WhatsApp. With it you can search and read your personal WhatsApp messages (including images, videos, documents, and audio), search your contacts, and send messages to individuals or groups — all through an AI agent like Claude.

It connects **directly to your personal WhatsApp account** via the WhatsApp web multidevice API (the [whatsmeow](https://github.com/tulir/whatsmeow) library). All messages are stored **locally** in a SQLite database and are only sent to the LLM when the agent accesses them through the tools — which you control.

![WhatsApp MCP](./example-use.png)

## What this fork adds

Compared to the original project, this fork adds:

- 🔔 **Native desktop notification (KDE Plasma / GNOME / any libnotify)** every time the bridge starts and becomes ready (`notify-send`, no new Go dependency).
- 🔄 **Migration to the current whatsmeow API** — `context.Context` parameters on `Download`, `sqlstore.New`, `GetFirstDevice`, `GetGroupInfo`, and `GetContact`, plus updated dependencies (whatsmeow, libsignal, go-sqlite3, protobuf). Fixes build/connection failures against the newer WhatsApp server versions.
- ⚙️ **systemd user service** to start automatically on boot and restart on failure (instructions below).

> ⚠️ **Security note:** the bridge's REST API (`:8080`) has **no authentication**, so it now listens on **`127.0.0.1` only** (loopback). It is not reachable from the network; the MCP server runs on the same machine and talks to it locally. If you need remote access, use an SSH tunnel or run the MCP server over SSH (stdio) instead of exposing the port.

> ⚠️ **About account bans:** whatsmeow is an **unofficial** client, and using it technically violates WhatsApp's Terms of Service. For personal, moderate use (reading chats, replying to contacts who already know you, human-like volume) the risk is low but real. **Avoid** bulk sending, aggressive automation, and messaging numbers that don't have you saved.

> ⚠️ Like many MCP servers, this one is subject to the [lethal trifecta](https://simonwillison.net/2025/Jun/16/the-lethal-trifecta/): prompt injection could lead to private data exfiltration.

## Installation

### Prerequisites

- **Go** (1.25+)
- **Python 3.6+**
- **UV** (Python package manager): `curl -LsSf https://astral.sh/uv/install.sh | sh`
- **Claude Desktop**, **Cursor**, or **Claude Code** (CLI)
- **FFmpeg** (_optional_) — only needed to send audio as WhatsApp voice messages (`.ogg` Opus format). With FFmpeg installed, the server auto-converts other formats.
- **libnotify** (`notify-send`) — for this fork's desktop notifications (already present on most KDE/GNOME setups).

### Steps

1. **Clone the fork**

   ```bash
   git clone https://github.com/hiltoncc03/whatsapp-mcp.git
   cd whatsapp-mcp
   ```

2. **Run the WhatsApp bridge**

   ```bash
   cd whatsapp-bridge
   go run main.go
   ```

   On first run, a **QR code** is shown in the terminal. Scan it with your phone's WhatsApp under **Settings → Linked devices → Link a device**.

   Once connected and ready, a **desktop notification** fires. After ~20 days you may need to re-authenticate.

3. **Connect to the MCP server**

   #### Claude Desktop / Cursor

   ```json
   {
     "mcpServers": {
       "whatsapp": {
         "command": "{{PATH_TO_UV}}",
         "args": [
           "--directory",
           "{{PATH_TO_SRC}}/whatsapp-mcp/whatsapp-mcp-server",
           "run",
           "main.py"
         ]
       }
     }
   }
   ```

   - `{{PATH_TO_UV}}`: output of `which uv`
   - `{{PATH_TO_SRC}}`: path where you cloned the repo (`pwd`)

   Save as `claude_desktop_config.json` (Claude Desktop) or `~/.cursor/mcp.json` (Cursor) and restart the app.

   #### Claude Code (CLI)

   ```bash
   claude mcp add whatsapp -- uv --directory /path/to/whatsapp-mcp/whatsapp-mcp-server run main.py
   ```

## Auto-start on boot (systemd — Linux)

This fork is designed to run as a service. Build the binary and create a user service:

1. **Build:**

   ```bash
   cd whatsapp-bridge
   go build -o whatsapp-bridge .
   ```

2. **Create** `~/.config/systemd/user/whatsapp-bridge.service`:

   ```ini
   [Unit]
   Description=WhatsApp MCP Bridge
   After=network-online.target
   Wants=network-online.target

   [Service]
   WorkingDirectory=%h/whatsapp-mcp/whatsapp-bridge
   ExecStart=%h/whatsapp-mcp/whatsapp-bridge/whatsapp-bridge
   Restart=on-failure
   RestartSec=5

   [Install]
   WantedBy=default.target
   ```

3. **Enable and start on boot:**

   ```bash
   systemctl --user daemon-reload
   systemctl --user enable --now whatsapp-bridge.service
   loginctl enable-linger "$USER"   # start on boot even without a graphical login
   ```

   Handy commands:

   ```bash
   systemctl --user status whatsapp-bridge      # state
   systemctl --user restart whatsapp-bridge     # restart
   journalctl --user -u whatsapp-bridge -f      # logs (e.g. QR on re-auth)
   ```

   > `WorkingDirectory` **must** point to `whatsapp-bridge/`, because the database is written to `store/` relative to that directory — and that's where the MCP server reads from. Starting from elsewhere makes the bridge write to a different `store/`, and the MCP server won't see the messages.

### Windows compatibility

`go-sqlite3` requires **CGO enabled**, which is off by default on Windows. Install a C compiler (via [MSYS2](https://www.msys2.org/), adding `ucrt64\bin` to `PATH`) and run:

```bash
cd whatsapp-bridge
go env -w CGO_ENABLED=1
go run main.go
```

Without this you'll see: `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work.`

## Architecture

The application has two components:

1. **Go WhatsApp Bridge** (`whatsapp-bridge/`): connects to WhatsApp's web API, authenticates via QR code, stores history in SQLite, and exposes a REST API on `:8080`.
2. **Python MCP Server** (`whatsapp-mcp-server/`): implements the Model Context Protocol, providing the standardized tools for Claude to interact with WhatsApp data.

### Data storage

- History lives in SQLite under `whatsapp-bridge/store/` (`messages.db` and `whatsapp.db`).
- Data is stored **in plaintext** — keep the directory permissions restricted to your user.

## MCP tools

- **search_contacts**: search contacts by name or phone number
- **list_messages**: retrieve messages with filters and context
- **list_chats**: list chats with metadata
- **get_chat**: info about a specific chat
- **get_direct_chat_by_contact**: find the direct chat with a contact
- **get_contact_chats**: list all chats involving a contact
- **get_last_interaction**: most recent message with a contact
- **get_message_context**: context around a specific message
- **send_message**: send a message to a phone number or group JID
- **send_file**: send a file (image, video, raw audio, document)
- **send_audio_message**: send audio as a voice message (requires `.ogg` Opus or FFmpeg installed)
- **download_media**: download a message's media and return the local path

### Media

**Sending:** use `send_file` for images/videos/documents and `send_audio_message` for voice messages (with FFmpeg, it auto-converts MP3/WAV to `.ogg` Opus).

**Downloading:** by default only media metadata is stored in the database. To access the file, use `download_media` with the `message_id` and `chat_jid` (shown when listing messages); it downloads and returns the local path.

## How it works

1. Claude sends requests to the Python MCP server
2. The MCP server queries the Go bridge / the SQLite database
3. The Go bridge accesses the WhatsApp API and keeps SQLite up to date
4. Data flows back through the chain to Claude
5. When sending, the flow goes Claude → MCP server → Go bridge → WhatsApp

## Troubleshooting

- **QR code not showing**: restart the bridge; check that your terminal can display QR codes.
- **Already logged in**: if the session is active, the bridge reconnects without showing a QR.
- **Device limit reached**: WhatsApp limits linked devices. Remove one under **Settings → Linked devices**.
- **Messages not loading**: after authenticating, history can take a few minutes to sync.
- **`list_chats` empty / MCP sees no messages**: almost always a path mismatch — the bridge must be started from inside `whatsapp-bridge/` (see the systemd note) so it writes to the same `store/` the MCP server reads.
- **WhatsApp out of sync**: delete `whatsapp-bridge/store/messages.db` and `whatsapp-bridge/store/whatsapp.db` and restart the bridge to re-authenticate.

## Credits

Fork of [`lharries/whatsapp-mcp`](https://github.com/lharries/whatsapp-mcp). Licensed under [MIT](./LICENSE).
