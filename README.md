# RTZ Server

[![Release](https://github.com/USA-RedDragon/rtz-server/actions/workflows/release.yaml/badge.svg)](https://github.com/USA-RedDragon/rtz-server/actions/workflows/release.yaml) [![License](https://badgen.net/github/license/USA-RedDragon/rtz-server)](https://github.com/USA-RedDragon/rtz-server/blob/main/LICENSE) [![go.mod version](https://img.shields.io/github/go-mod/go-version/USA-RedDragon/rtz-server.svg)](https://github.com/USA-RedDragon/rtz-server) [![coverage](https://raw.githubusercontent.com/USA-RedDragon/rtz-server/main/.github/badges/coverage.svg)](https://github.com/USA-RedDragon/rtz-server/actions)

![RTZ logo](./hack/hero.png)

An implementation of the Comma.ai API service for self-hosted folks. Pronounced like "Routes".

> [!WARNING]
> This is considered under _ACTIVE DEVELOPMENT_ until v1.0.0 or later.
> Any v0 releases are considered pre-alpha and have wildly breaking changes that may require complete wipes of the database and/or uploads.

## Benefits

- Doesn't track you
- Keeps your video data to yourself
- Simple to install
- Small binary (less than 15MB), self-contained
- Store data long-term
- Not paid, only self-hosting costs
- Same frontend/PWA you're used to (Connect's frontend is open-source, thanks Comma!)
- Doesn't store emails
- Optional high availability (HA) setup with NATS

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

See [Configuration](#configuration).

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
sed -i "7i export MAPS_HOST=\"$URL\"" launch_env.sh
sed -i '8i # end of rtz-server configuration\n' launch_env.sh

# Removes hard-coded Comma API URL
# Some versions of OpenPilot have removed navd, so we need to check for its existence
if test -f selfdrive/navd/navd.py; then
  sed -i 's#self.mapbox_host = "https://maps.comma.ai"#self.mapbox_host = os.getenv("MAPS_HOST", "https://maps.comma.ai")#' selfdrive/navd/navd.py
fi

# Now reboot
sudo reboot
```

## Configuration

The server reads its settings from, lowest to highest priority, the defaults below, a YAML config file, environment variables and command line flags. The config file is `config.yaml` in the working directory, or the path given with `--config` (`-c`). [`config.example.yaml`](config.example.yaml) lists every option. List values in environment variables and flags are comma-separated, i.e. `HTTP__CORS_HOSTS="https://a.example.com,https://b.example.com"`.

<!-- configulator:begin -->

| Key                                                | Type           | Default      | Environment                                           | Flag                                                 | Description                                                                                                                                            |
|----------------------------------------------------|----------------|--------------|-------------------------------------------------------|------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|
| `http.ipv4_host`                                   | string         | `0.0.0.0`    | `HTTP__IPV4_HOST`                                     | `--http.ipv4_host`                                   | HTTP server IPv4 host                                                                                                                                  |
| `http.ipv6_host`                                   | string         | `::`         | `HTTP__IPV6_HOST`                                     | `--http.ipv6_host`                                   | HTTP server IPv6 host                                                                                                                                  |
| `http.port`                                        | integer        | `8080`       | `HTTP__PORT`                                          | `--http.port`                                        | HTTP server port, shared by IPv4 and IPv6                                                                                                              |
| `http.tracing.enabled`                             | boolean        |              | `HTTP__TRACING__ENABLED`                              | `--http.tracing.enabled`                             | Enable OpenTelemetry tracing                                                                                                                           |
| `http.tracing.otlp_endpoint`                       | string         |              | `HTTP__TRACING__OTLP_ENDPOINT`                        | `--http.tracing.otlp_endpoint`                       | OpenTelemetry collector endpoint, required when tracing is enabled                                                                                     |
| `http.backend_url`                                 | string         |              | `HTTP__BACKEND_URL`                                   | `--http.backend_url`                                 | Public URL of this server (required)                                                                                                                   |
| `http.pprof.enabled`                               | boolean        |              | `HTTP__PPROF__ENABLED`                                | `--http.pprof.enabled`                               | Enable pprof                                                                                                                                           |
| `http.trusted_proxies`                             | list of string |              | `HTTP__TRUSTED_PROXIES`                               | `--http.trusted_proxies`                             | IP addresses or CIDR ranges of reverse proxies trusted to set X-Forwarded-For                                                                          |
| `http.metrics.ipv4_host`                           | string         | `127.0.0.1`  | `HTTP__METRICS__IPV4_HOST`                            | `--http.metrics.ipv4_host`                           | Prometheus metrics server IPv4 host                                                                                                                    |
| `http.metrics.ipv6_host`                           | string         | `::1`        | `HTTP__METRICS__IPV6_HOST`                            | `--http.metrics.ipv6_host`                           | Prometheus metrics server IPv6 host                                                                                                                    |
| `http.metrics.port`                                | integer        | `8081`       | `HTTP__METRICS__PORT`                                 | `--http.metrics.port`                                | Prometheus metrics server port, shared by IPv4 and IPv6                                                                                                |
| `http.metrics.enabled`                             | boolean        |              | `HTTP__METRICS__ENABLED`                              | `--http.metrics.enabled`                             | Enable the Prometheus metrics server                                                                                                                   |
| `http.cors_hosts`                                  | list of string |              | `HTTP__CORS_HOSTS`                                    | `--http.cors_hosts`                                  | Hosts allowed by CORS                                                                                                                                  |
| `persistence.database.driver`                      | string         | `sqlite`     | `PERSISTENCE__DATABASE__DRIVER`                       | `--persistence.database.driver`                      | Database driver, one of: sqlite, mysql, postgres                                                                                                       |
| `persistence.database.database`                    | string         | `rtz.db`     | `PERSISTENCE__DATABASE__DATABASE`                     | `--persistence.database.database`                    | Path to the SQLite database file, or the database name for other drivers                                                                               |
| `persistence.database.username`                    | string         |              | `PERSISTENCE__DATABASE__USERNAME`                     | `--persistence.database.username`                    | Database username                                                                                                                                      |
| `persistence.database.password`                    | string         |              | `PERSISTENCE__DATABASE__PASSWORD`                     | `--persistence.database.password`                    | Database password (secret)                                                                                                                             |
| `persistence.database.host`                        | string         |              | `PERSISTENCE__DATABASE__HOST`                         | `--persistence.database.host`                        | Database host, required for mysql and postgres                                                                                                         |
| `persistence.database.port`                        | integer        |              | `PERSISTENCE__DATABASE__PORT`                         | `--persistence.database.port`                        | Database port, 0 uses the driver's default                                                                                                             |
| `persistence.database.extra_parameters`            | string         |              | `PERSISTENCE__DATABASE__EXTRA_PARAMETERS`             | `--persistence.database.extra_parameters`            | Extra parameters passed to the database driver                                                                                                         |
| `persistence.uploads.driver`                       | string         | `filesystem` | `PERSISTENCE__UPLOADS__DRIVER`                        | `--persistence.uploads.driver`                       | Storage driver for uploaded videos and driving logs, one of: filesystem, s3                                                                            |
| `persistence.uploads.filesystem_options.directory` | string         | `uploads/`   | `PERSISTENCE__UPLOADS__FILESYSTEM_OPTIONS__DIRECTORY` | `--persistence.uploads.filesystem_options.directory` | Filesystem uploads directory, created if it does not exist                                                                                             |
| `persistence.uploads.s3_options.region`            | string         |              | `PERSISTENCE__UPLOADS__S3_OPTIONS__REGION`            | `--persistence.uploads.s3_options.region`            | S3 region. Credentials come from the standard AWS environment variables, the AWS CLI config or an IAM role                                             |
| `persistence.uploads.s3_options.bucket`            | string         |              | `PERSISTENCE__UPLOADS__S3_OPTIONS__BUCKET`            | `--persistence.uploads.s3_options.bucket`            | S3 bucket                                                                                                                                              |
| `persistence.uploads.s3_options.endpoint`          | string         |              | `PERSISTENCE__UPLOADS__S3_OPTIONS__ENDPOINT`          | `--persistence.uploads.s3_options.endpoint`          | Custom S3 endpoint, which switches to path-style addressing                                                                                            |
| `registration.enabled`                             | boolean        |              | `REGISTRATION__ENABLED`                               | `--registration.enabled`                             | Enable user registration                                                                                                                               |
| `auth.google.enabled`                              | boolean        |              | `AUTH__GOOGLE__ENABLED`                               | `--auth.google.enabled`                              | Enable Google OAuth                                                                                                                                    |
| `auth.google.client_id`                            | string         |              | `AUTH__GOOGLE__CLIENT_ID`                             | `--auth.google.client_id`                            | Google OAuth client ID                                                                                                                                 |
| `auth.google.client_secret`                        | string         |              | `AUTH__GOOGLE__CLIENT_SECRET`                         | `--auth.google.client_secret`                        | Google OAuth client secret (secret)                                                                                                                    |
| `auth.github.enabled`                              | boolean        |              | `AUTH__GITHUB__ENABLED`                               | `--auth.github.enabled`                              | Enable GitHub OAuth                                                                                                                                    |
| `auth.github.client_id`                            | string         |              | `AUTH__GITHUB__CLIENT_ID`                             | `--auth.github.client_id`                            | GitHub OAuth client ID                                                                                                                                 |
| `auth.github.client_secret`                        | string         |              | `AUTH__GITHUB__CLIENT_SECRET`                         | `--auth.github.client_secret`                        | GitHub OAuth client secret (secret)                                                                                                                    |
| `auth.custom.enabled`                              | boolean        |              | `AUTH__CUSTOM__ENABLED`                               | `--auth.custom.enabled`                              | Enable custom OAuth                                                                                                                                    |
| `auth.custom.client_id`                            | string         |              | `AUTH__CUSTOM__CLIENT_ID`                             | `--auth.custom.client_id`                            | Custom OAuth client ID                                                                                                                                 |
| `auth.custom.client_secret`                        | string         |              | `AUTH__CUSTOM__CLIENT_SECRET`                         | `--auth.custom.client_secret`                        | Custom OAuth client secret (secret)                                                                                                                    |
| `auth.custom.token_url`                            | string         |              | `AUTH__CUSTOM__TOKEN_URL`                             | `--auth.custom.token_url`                            | Custom OAuth token URL                                                                                                                                 |
| `auth.custom.user_url`                             | string         |              | `AUTH__CUSTOM__USER_URL`                              | `--auth.custom.user_url`                             | Custom OAuth user URL                                                                                                                                  |
| `jwt.secret`                                       | string         |              | `JWT__SECRET`                                         | `--jwt.secret`                                       | JWT signing secret (required) (secret)                                                                                                                 |
| `mapbox.secret_token`                              | string         |              | `MAPBOX__SECRET_TOKEN`                                | `--mapbox.secret_token`                              | Mapbox secret token (required) (secret)                                                                                                                |
| `mapbox.public_token`                              | string         |              | `MAPBOX__PUBLIC_TOKEN`                                | `--mapbox.public_token`                              | Mapbox public token (required)                                                                                                                         |
| `nats.enabled`                                     | boolean        |              | `NATS__ENABLED`                                       | `--nats.enabled`                                     | Enable NATS. Required when running more than one instance, such as blue/green deployments, so websockets reach the instance the device is connected to |
| `nats.url`                                         | string         |              | `NATS__URL`                                           | `--nats.url`                                         | NATS URL                                                                                                                                               |
| `nats.token`                                       | string         |              | `NATS__TOKEN`                                         | `--nats.token`                                       | NATS token (secret)                                                                                                                                    |
| `parallel_log_parsers`                             | integer        | `4`          | `PARALLEL_LOG_PARSERS`                                | `--parallel_log_parsers`                             | Number of parallel log parsers                                                                                                                         |
| `log_level`                                        | string         | `info`       | `LOG_LEVEL`                                           | `--log_level`                                        | Log level, one of: debug, info, warn, error                                                                                                            |

<!-- configulator:end -->

## TODOs (in order of priority)

- [ ] Parsing uploaded segments and routes (partially done)
- [ ] Stat tracking
- [ ] Useradmin
- [ ] Add more documentation
- [ ] Add more tests
- [ ] Document deployment
- [ ] Tool to copy data from Comma's servers
- [ ] SSH forwarding

## Wants

I would really like to have an opt-in configuration option that allows for data to be reuploaded to Comma's Connect servers under the user's previous dongle ID. That way users can still contribute to the larger dataset for training if they so choose. If you work at Comma and can help find someone who can bless this idea, please reach out to me. I don't want to do this without their permission.
