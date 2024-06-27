# connect-server

[![Release](https://github.com/USA-RedDragon/connect-server/actions/workflows/release.yaml/badge.svg)](https://github.com/USA-RedDragon/connect-server/actions/workflows/release.yaml) [![License](https://badgen.net/github/license/USA-RedDragon/connect-server)](https://github.com/USA-RedDragon/connect-server/blob/master/LICENSE) [![go.mod version](https://img.shields.io/github/go-mod/go-version/USA-RedDragon/connect-server.svg)](https://github.com/USA-RedDragon/connect-server) [![GoReportCard](https://goreportcard.com/badge/github.com/USA-RedDragon/connect-server)](https://goreportcard.com/report/github.com/USA-RedDragon/connect-server) [![codecov](https://codecov.io/gh/USA-RedDragon/connect-server/graph/badge.svg?token=6ASKMAKOZE)](https://codecov.io/gh/USA-RedDragon/connect-server)

An implementation of the Comma.ai Connect service for self-hosted folks.

## Configuration

The service is configured via environment variables, a configuration YAML file, or command line flags. The [`config.example.yaml`](config.example.yaml) file shows the available configuration options. The command line flags match the schema of the YAML file, i.e. `--http.cors_hosts='0.0.0.0'` would equate to `http.cors_hosts: ["0.0.0.0"]`. Environment variables are in the same format, however they are uppercase and replace hyphens with underscores and dots with double underscores, i.e. `HTTP__CORS_HOSTS="0.0.0.0"`.
