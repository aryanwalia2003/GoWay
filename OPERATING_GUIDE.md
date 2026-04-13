# GoWay Operating Guide

This guide describes how to set up, deploy, and monitor the GoWay microservice using the provided automation scripts.

## 🚀 Quick Start

1.  **Run Setup**: Initialize the environment and check requirements.
    ```bash
    ./scripts/setup.sh
    ```
2.  **Configure Environment**: Edit the generated `.env` file with your specific settings (API keys, ports, etc.).
3.  **Deploy**: Build and start the service.
    ```bash
    ./scripts/deploy.sh up
    ```
4.  **Verify**: Ensure the service is healthy.
    ```bash
    ./scripts/deploy.sh check
    ```

---

## 📂 Script Reference

### `scripts/setup.sh`
Used for the first-time setup or whenever requirements change.
- **Checks**: Verifies `docker`, `docker-compose`, and `go` are installed.
- **Initialization**: Creates a `.env` file from `.env.example` (if not already present).
- **Validation**: Checks for the existence of the `templates/` directory and critical assets like the Zippee logo.

### `scripts/deploy.sh {command}`
The primary tool for managing the service lifecycle.
- `up`: Builds the Docker image and starts the container in detached mode.
- `down`: Stops and removes the containers and networks.
- `restart`: Quickly restarts the running service.
- `logs`: Follows the application logs in real-time.
- `check`: Performs a health check on the `/readyz` and `/livez` endpoints.

### Monitoring & Analysis
- **`scripts/monitor.sh`**: Runs in the background (or a side terminal) to sample container CPU/Memory and write to `stats.csv`.
- **`scripts/analyze.sh`**: Processes logs to identify total requests, average/min/max latency, and overall throughput.
- **`scripts/zombie_check.sh`**: Scans the container's process tree for orphaned rendering sub-processes that might be leaking resources.

---


## 🔍 Monitoring & Health Checks

The service provides standard Kubernetes-style probes for observability:

- **Liveness (`/livez`)**: Returns `200 OK` if the process is running.
- **Readiness (`/readyz`)**: Returns `200 OK` only if:
    1.  The template directory is accessible.
    2.  The Zippee logo is successfully loaded into memory.
- **General Health (`/healthz`)**: Standard startup/health check.

You can monitor these manually using `curl` or by running `./scripts/deploy.sh check`.

---

## 🛠 Troubleshooting

- **Port Conflicts**: If the deployment fails to bind, check if the `PORT` in your `.env` is already in use on the host.
- **Logo Issues**: If labels are rendering without the logo, ensure the logo file path in `templates/` matches what the assembler expects.
- **Auth Errors**: Ensure your requests include the `X-API-Key` or `Authorization` header matching one of the keys in `API_KEYS`.
