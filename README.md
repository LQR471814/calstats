# calutil

> The missing tools for managing calendars at scale.

![demo.png](./docs/demo.png)

## Features

- **Statistics:** Visualize how your time is spent.
- **Bulk edit:** Edit calendar events in bulk.
- **Version history:** View changes to your calendar over time.

## Configuration

```json5
// config.json5
[
	{
		server: {
			url: "https://<caldav_server_host>/<username>",
			insecure: true, // enable if you want to ignore SSL issues
			username: "<username>",
			password: "<password>",
		},
		calendars: ["<calendar_name>", ...]
	},
	...
]
```

## Usage

```sh
./calutil --config <path/to/config.json5> serve
```

## Build

```sh
cd ui && pnpm build
go build
```

