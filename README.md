<h1 align="center"> in⧕idents </h1>

Inxidents is a minimal configuration, open-source incident management software with alerts and dashboard for your HTTP/S services.

**Current Features:**
- Real-time (SSE) Health Dashboard of your services. Perfect for office screens or similar environments.
- Slack Alerts whenever a service goes down.
- Visually see when the next healthcheck will occurr (the white progresbar)
- Small project with simple configuration. Easy to hack and extend for your needs.

**Upcoming features:**
- Acknowledgement Button for down services so alerts stop. 
- Add "expectedString" configuration for more functional testing.
- Recovered Alert
- ... ideas and suggestions are welcome

# Demo
[Click for Demo Dashboard](https://incidents.fly.dev/)

https://github.com/piqoni/incidents/assets/3144671/5a6f1466-ff29-455d-b0af-dd60f11b8d8b

# Installation / Deployment
1. ```cp config.dev.yaml config.yaml```
2. Change config.yaml accordingly and add your services:
Sample service: 
```
- name: Google
  endpoint: https://www.google.com
  frequency: 1m
  expectedCode: 200
```
- **Name**: Name of service, currently it needs to be unique for each service you check. 
- **Endpoint**: HTTP/S endpoint
- **Frequency**:  Frequency of the health check, examples: "300ms", "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".


## Deploy on fly.io
1. Install [flytcl](https://fly.io/docs/hands-on/install-flyctl/)
2. Run ```flyctl launch```(answer no to DB or Volume creations)
3. Run ```flyctl deploy``` to deploy

## Other deployment methods
You can deploy it via docker as it is containarized or if you get the self-contained binary, you can use systemd to keep the process running.

TODO: Needs more documentation here.
