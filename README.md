# Hackatime CLI

[![Tests](https://img.shields.io/github/actions/workflow/status/hackclub/hackatime-cli/on_push.yml?branch=develop&label=tests)](https://github.com/hackclub/hackatime-cli/actions)
[![Downloads](https://img.shields.io/github/downloads/hackclub/hackatime-cli/total?color=007ec6)](https://github.com/hackclub/hackatime-cli/releases)

Command line interface used by [Hackatime][hackatime] editor plugins, such as
[vscode-hackatime][vscode-hackatime].

hackatime-cli is a fork of [wakatime-cli][wakatime-cli]. It speaks the same WakaTime-compatible
API and sends heartbeats to Hackatime by default.

## Usage

Normally you don't need to use hackatime-cli directly unless you're building an editor plugin.

hackatime-cli reads the same [INI][usage] config file as WakaTime plugins, so existing setups keep
working:

`$WAKATIME_HOME/.wakatime.cfg`

Without an explicit config path or `WAKATIME_HOME`, hackatime-cli also supports an
[XDG config fallback](USAGE.md#config-file) when `~/.wakatime.cfg` does not exist.

hackatime-cli writes JSONL logs to:

`$WAKATIME_HOME/.wakatime/wakatime.log`

Set `debug = true` in your config file to enable verbose logs.

See [Usage][usage] for more details.

## Differences from wakatime-cli

- The binary and release assets are named `hackatime-cli-<os>-<arch>`.
- `api_url` defaults to `https://hackatime.hackclub.com/api/hackatime/v1`.
- There is no hardcoded IP fallback when DNS resolution fails.

The config file, `~/.wakatime/` folder, `WAKATIME_*` environment variables, `.wakatime-project`
files and the `wakatime/<version>` user agent prefix are unchanged, so hackatime-cli can share a
machine with wakatime-cli and with plugins that still use it.

## Contributing

Pull requests and issues are welcome!
See [Contributing][contributing] for more details.

## Troubleshooting

See [Troubleshooting][troubleshooting] for more details.

Many thanks to the WakaTime team and all [contributors][authors] to wakatime-cli.

[hackatime]: https://hackatime.hackclub.com
[vscode-hackatime]: https://github.com/hackclub/vscode-hackatime
[wakatime-cli]: https://github.com/wakatime/wakatime-cli
[usage]: USAGE.md
[contributing]: CONTRIBUTING.md
[troubleshooting]: TROUBLESHOOTING.md
[authors]: AUTHORS
