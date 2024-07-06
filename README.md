# connect-server

[![Release](https://github.com/USA-RedDragon/connect-server/actions/workflows/release.yaml/badge.svg)](https://github.com/USA-RedDragon/connect-server/actions/workflows/release.yaml) [![License](https://badgen.net/github/license/USA-RedDragon/connect-server)](https://github.com/USA-RedDragon/connect-server/blob/master/LICENSE) [![go.mod version](https://img.shields.io/github/go-mod/go-version/USA-RedDragon/connect-server.svg)](https://github.com/USA-RedDragon/connect-server) [![GoReportCard](https://goreportcard.com/badge/github.com/USA-RedDragon/connect-server)](https://goreportcard.com/report/github.com/USA-RedDragon/connect-server) [![codecov](https://codecov.io/gh/USA-RedDragon/connect-server/graph/badge.svg?token=6ASKMAKOZE)](https://codecov.io/gh/USA-RedDragon/connect-server)

An implementation of the Comma.ai API service for self-hosted folks.

> [!WARNING]
> This is considered under _ACTIVE DEVELOPMENT_ until v1.0.0 or later.
> Any v0 releases are considered pre-alpha and have wildly breaking changes that may require complete wipes of the database and/or uploads.

## Benefits

- Doesn't track you
- Keeps your video data to yourself
- Simple to install
- Small binary (less than 10MB), self-contained
- Store data long-term
- Not paid, only self-hosting costs
- Same frontend/PWA you're used to (Connect's frontend is open-source, thanks Comma!)
- Doesn't store emails or passwords

## Configuration

The service is configured via environment variables, a configuration YAML file, or command line flags. The [`config.example.yaml`](config.example.yaml) file shows the available configuration options. The command line flags match the schema of the YAML file, i.e. `--http.cors_hosts='0.0.0.0'` would equate to `http.cors_hosts: ["0.0.0.0"]`. Environment variables are in the same format, however they are uppercase and replace hyphens with underscores and dots with double underscores, i.e. `HTTP__CORS_HOSTS="0.0.0.0"`.

## TODOs

- [ ] Stat tracking
- [ ] Parsing uploaded segments and routes
- [ ] Get/set next navigation
- [ ] Add more documentation
- [ ] Add more tests
- [ ] Allow custom IDPs for sign in
- [ ] Document deployment
- [ ] Useradmin
- [ ] Rename the project

## Frotend TODOs

- [ ] User-configurable MAPBOX_TOKEN
- [ ] Replace HERE API with Google Maps API
- [ ] Remove Comma trademarked assets
- [ ] Demo mode

## Wants

I would really like to have an opt-in configuration option that allows for data to be reuploaded to Comma's Connect servers under the user's previous dongle ID. That way users can still contribute to the larger dataset for training if they so choose. If you work at Comma and can help find someone who can bless this idea, please reach out to me. I don't want to do this without their permission.
