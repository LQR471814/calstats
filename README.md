# schedule-statistics

> Visualize the usage of your time given a schedule.

![demo.png](./docs/demo.png)

## Configuration

```json5
// config.json5
{
	server: {
		url: "https://<caldav_server_host>/<username>",
		insecure: true, // enable if you want to ignore SSL issues
		username: "<username>",
		password: "<password>",
	},
	calendars: ["<calendar_name>", ...]
}
```

## Usage

```sh
./schedule-statistics -config <path/to/config.json5>
```

## Build

```sh
cd ui && pnpm build
```

