# RTZ Server

[![Release](https://github.com/USA-RedDragon/rtz-server/actions/workflows/release.yaml/badge.svg)](https://github.com/USA-RedDragon/rtz-server/actions/workflows/release.yaml) [![License](https://badgen.net/github/license/USA-RedDragon/rtz-server)](https://github.com/USA-RedDragon/rtz-server/blob/master/LICENSE) [![go.mod version](https://img.shields.io/github/go-mod/go-version/USA-RedDragon/rtz-server.svg)](https://github.com/USA-RedDragon/rtz-server) [![GoReportCard](https://goreportcard.com/badge/github.com/USA-RedDragon/rtz-server)](https://goreportcard.com/report/github.com/USA-RedDragon/rtz-server) [![codecov](https://codecov.io/gh/USA-RedDragon/rtz-server/graph/badge.svg?token=6ASKMAKOZE)](https://codecov.io/gh/USA-RedDragon/rtz-server)

An implementation of the Comma.ai API service for self-hosted folks. Pronounced like "Routes".

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
- Doesn't store emails
- Optional high availability (HA) setup with Redis

## Emulated Services

This project emulates more than just the Comma.ai API. It also emulates portions of the following services:

- billing.comma.ai (These routes are stubbed out to report an active Comma Prime membership to OpenPilot)
- maps.comma.ai (OpenPilot calls this to get routing data)
- Athena (This is the websocket service that OpenPilot uses for JSON RPC)
- (eventually) useradmin.comma.ai

## Setup

### Frontend

You will need to run the Comma Connect frontend to get full functionality. You can find the frontend at [rtz-frontend](https://github.com/USA-RedDragon/rtz-frontend). The frontend is the same as Comma's, but with the branding and tracking removed. You can also use the official frontend, but you will need to modify the `config.js` file to point to your server.

### Server Configuration

The service is configured via environment variables, a configuration YAML file, or command line flags. The [`config.example.yaml`](config.example.yaml) file shows the available configuration options. The command line flags match the schema of the YAML file, i.e. `--http.cors_hosts='0.0.0.0'` would equate to `http.cors_hosts: ["0.0.0.0"]`. Environment variables are in the same format, however they are uppercase and replace hyphens with underscores and dots with double underscores, i.e. `HTTP__CORS_HOSTS="0.0.0.0"`.

### Device Configuration

To configure a device running OpenPilot, you will need SSH access to the device. Documentation for this is available in [Comma's documentation](https://docs.comma.ai/how-to/connect-to-comma/#ssh). Once you have SSH access, you can run the following commands while logged into the device to configure it to use your self-hosted server:

> [!WARNING]
> This process will have to be followed every time OpenPilot is updated.

Replace `URL` and `WEBSOCKET_URL` with your server's URL. It can work over HTTP as well, however I only recommend this when your device is on a Wifi network, if you have a SIM you'll want to expose this service behind a load balancer with a valid SSL certificate. If you don't have a valid SSL certificate, you can use [Let's Encrypt](https://letsencrypt.org/) to get one for free.

```sh
URL="https://your-server.com" # Replace this with your server's URL
WEBSOCKET_URL="wss://your-server.com" # Replace this with your server's URL

cd /data/openpilot

# Adds the rtz-server configuration to the launch_env.sh file
sed -i '3i # rtz-server configuration, comment or remove the following lines to revert back to stock' launch_env.sh
sed -i '4i # comment or remove the following lines to revert back to stock' launch_env.sh
sed -i "5i export ATHENA_HOST=\"$WEBSOCKET_URL\"" launch_env.sh
sed -i "6i export API_HOST=\"$URL\"" launch_env.sh
sed -i "7i export COMMA_MAPS_HOST=\"$URL\"" launch_env.sh
sed -i '8i # end of rtz-server configuration\n' launch_env.sh

# Removes hard-coded Comma API URL
# Some versions of OpenPilot have removed navd, so we need to check for its existence
if test -f selfdrive/navd/navd.py; then
  sed -i 's#self.mapbox_host = "https://maps.comma.ai"#self.mapbox_host = os.getenv("COMMA_MAPS_HOST", "https://maps.comma.ai")#' selfdrive/navd/navd.py
fi

# Now reboot
sudo reboot
```

## TODOs (in order of priority)

- [ ] Parsing uploaded segments and routes
- [ ] Stat tracking
- [ ] Useradmin
- [ ] Add more documentation
- [ ] Add more tests
- [ ] Document deployment
- [ ] Demo mode

## Frotend TODOs

- [ ] Replace HERE API key
- [ ] Demo mode

## Wants

I would really like to have an opt-in configuration option that allows for data to be reuploaded to Comma's Connect servers under the user's previous dongle ID. That way users can still contribute to the larger dataset for training if they so choose. If you work at Comma and can help find someone who can bless this idea, please reach out to me. I don't want to do this without their permission.
