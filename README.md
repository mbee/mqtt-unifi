# mqtt-unifi

mqtt-unifi publishes on mqtt information from clients connected to a unifi controller

## usage

there's no flag, all parameters are set by environment variables

## variables

variable | default | mandatory | description
---------|---------|-----------|------------
`MQTT_URL`      | localhost:1883 | [ ] | url for the mqtt server
`MQTT_LOGIN`    |                | [ ] | mqtt server login. Empty if no login
`MQTT_PASSWORD` |                | [ ] | mqtt server password. Empty if no password
`UNIFI_HOST`    |                | [x] | unifi server name
`UNIFI_USER`    |                | [x] | unifi server login
`UNIFI_PASS`    |                | [x] | unifi server password
`UNIFI_VERSION` | 5              | [ ] | unifi api version
`UNIFI_PORT`    | 8443           | [ ] | unifi server port
`UNIFI_SITE_ID` | default        | [ ] | unifi site id
`UNITI_DELAY`   | 3              | [ ] | refresh rate (seconds)

## events

### subscribe

mqtt-unifi will listen for the following events:

input topic | input message | description
------------|---------------|------------
`mqtt-unifi/get/host/<mac address>` | `{}` |  Will answer with mqtt-unifi/status/host/<mac address>

### publish

mqtt-unifi will publish the following events:

topic | message | description
------|---------|------------
`mqtt-unifi/status/host/<mac address>` | default host payload (see below) | sent back after a mqtt-unifi/get/host/#
`mqtt-unifi/<mac address>/new` | default host payload (see below) | sent each time a new host appears
`mqtt-unifi/<mac address>/new` | default host payload (see below) | sent each time a new host appears

### payload

if the host requested by `mqtt-unifi/get/host/<mac address>` does not exist, mqtt-unit will answer:

```json
{
	"mac": "<mac address>",
	"name": "",
	"ip": "",
	"ap": "",
	"channel": "",
	"essid": "",
}
```

otherwise:

```json
{
	"mac": "<mac address>",
	"name": "OnePlus4",
	"ip": "192.168.0.2",
	"ap": "",
	"channel": "6",
	"essid": "freebox",
 }
```
