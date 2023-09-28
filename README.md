<h1 align="center"> in⧕idents </h1>

# intro
In⧕idents is an open-source incident management software. It  supports:
- Real-time (SSE) Health Dashboard of your services. Perfect for offices or similar environments.
- Slack Alerts whenever a service goes down or when in⧕idents itself fails to run.

Upcoming features:
- Acknowledgement of Incidents so Alerts stop
- Add "expectedString" on an endpoint for more functional testing.

# Demo
[demo dashboard](https://incidents.fly.dev/)
https://github.com/piqoni/incidents/assets/3144671/5a6f1466-ff29-455d-b0af-dd60f11b8d8b

# Installation / Deployment
## Deploy on fly.io
1. Install [flytcl](https://fly.io/docs/hands-on/install-flyctl/)
2. Run ```flyctl launch```(answer no to DB or Volume creations)
3. Run ```flyctl deploy``` to deploy

## Other deployment methods
You can deploy it via docker as it is containarized or if you get the self-contained binary, you can use systemd to keep the process running.

TODO: Needs more documentation here.
