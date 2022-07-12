# Extensions dashboard

## CLI

### Usage:
Before start, you have to get credentials to the stores API and put them in the environment variables.

```dotenv
CHROME_CLIENT_ID=<client_id>
CHROME_CLIENT_SECRET=<client_secret>
CHROME_REFRESH_TOKEN=<refresh_token>

FIREFOX_CLIENT_ID=<client_id>
FIREFOX_CLIENT_SECRET=<client_secret>

EDGE_CLIENT_ID=<client_id>
EDGE_CLIENT_SECRET=<client_secret>
EDGE_ACCESS_TOKEN_URL=<access_token_url>
```

After that, you can use the CLI.

```sh
extdash [global options] command [command options] [arguments...]
```

#### Commands:
```
- status   returns extension info
- insert   uploads extension to the store
- update   uploads new version of extension to the store
- publish  publishes extension to the store
- sign     signs extension in the store
- help, h  Shows a list of commands or help for one command
```

#### Examples:

To get status of the extension in the Chrome store:
```sh
./extdash status chrome -app <app_id>
```

To upload new extension to the Mozilla store:
```sh
./extdash insert firefox -f /path/to/file -s /path/to/source
```

## Planned features
- [ ] create CLI to deploy to the stores
  - [x] chrome
  - [x] mozilla
  - [x] edge
  - [ ] opera
  - [ ] static
- [ ] get publish status for extensions from storage (published, draft, on review)
  - [ ] subscribe on status change via email or slack 
- [ ] collect stats from storage

## Planned improvements
- [ ] setup sentry to collect errors
