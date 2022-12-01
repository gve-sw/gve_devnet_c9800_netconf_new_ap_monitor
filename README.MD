# Catalyst 9800 NETCONF New AP Monitor

This repo contains example code to demonstrate monitoring & notification of new Catalyst Access Points (AP) that are connected to a 9800 series wireless controller (WLC).

This code will:

- Establish a NETCONF push subscription to a list of 9800 WLCs
- Listen for new subscription messages, which contain a list of joined APs
- Store known APs in a MongoDB instance
- Publish new AP MAC addresses to a target MQTT broker

**Note:** This is a companion app to the [NETCONF AP provisioning](https://github.com/gve-sw/gve_devnet_c9800_netconf_ap_provisioning) tool. The provisioning tool will subscribe to MQTT messages published by this app, and automatically provision AP tags to the WLC.

## Contacts

- Matt Schmitz (mattsc@cisco.com)

## Solution Components

- Cisco Catalyst Wireless Access Points & Wireless Lan Controllers
- MongoDB
- MQTT Broker

## Installation/Configuration

**1 - Clone repo:**

```bash
git clone <repo_url>
```

**2 - Provide Config file**

This code relies on a JSON configuration file (`config.json`) to provide the required WLC, MongoDB, and MQTT targets.

A sample configuration file has been provided and uses the format below:

```
{
    "wireless-controllers": [
        {
            "name": "",
            "port": 830
        }
    ],
    "netconf": {
        "xpath-filter": "/wireless-ap-global-oper:ap-global-oper-data/ap-join-stats/ap-join-info/is-joined",
        "subscription-period": 1000
    },
    "mqtt": {
        "broker": "",
        "port": 1883,
        "client-id": "go_mqtt_client",
        "topic":"wireless/ap"
    },
    "mongo": {
        "url":"mongodb://"
    }
}
```

**3 - Provide WLC Credentials:**

WLC login credentials are provided as environment variables:

```
export WLC_USER=
export WLC_PASSWORD=
```

**4 - Build executable:**

```bash
go build -o netconf-ap-monitor
```

## Usage

Run the application with the following command:

```
./netconf-ap-monitor
```

## Docker

A `Dockerfile` has been provided for easier deployment of this application. The container can be built and deployed using the following steps:

**1 - Clone repo:**

```bash
git clone <repo_url>
```

**2 - Build the container:**

```
docker build --tag netconf-ap-monitor .
```

**3 - Run the container:**

```
docker run -e WLC_USER=<user> -e WLC_PASSWORD=<password> -v <path-to-config.json>:/app/config.json -d netconf-ap-monitor
```

# Screenshots

**Example of app execution:**

![/IMAGES/app-execution.png](/IMAGES/app-execution.png)

**Stored APs in MongoDB:**

![/IMAGES/mongodb.png](/IMAGES/mongodb.png)

**Message sent to MQTT Broker:**

![/IMAGES/mqtt.png](/IMAGES/mqtt.png)

### LICENSE

Provided under Cisco Sample Code License, for details see [LICENSE](LICENSE.md)

### CODE_OF_CONDUCT

Our code of conduct is available [here](CODE_OF_CONDUCT.md)

### CONTRIBUTING

See our contributing guidelines [here](CONTRIBUTING.md)

#### DISCLAIMER

<b>Please note:</b> This script is meant for demo purposes only. All tools/ scripts in this repo are released for use "AS IS" without any warranties of any kind, including, but not limited to their installation, use, or performance. Any use of these scripts and tools is at your own risk. There is no guarantee that they have been through thorough testing in a comparable environment and we are not responsible for any damage or data loss incurred with their use.
You are responsible for reviewing and testing any scripts you run thoroughly before use in any non-testing environment.
