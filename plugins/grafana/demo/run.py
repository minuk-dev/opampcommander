#!/usr/bin/env python3
"""Run a disposable local Commander + real collector demo."""
import json
import os
from pathlib import Path
import subprocess
import sys
import time
import urllib.error
import urllib.request

PLUGIN = Path(__file__).resolve().parents[1]
ROOT = PLUGIN.parents[1]
STATE = PLUGIN / '.demo'
COMPOSE = ['docker', 'compose', '-f', 'docker-compose.yaml', '-f', 'compose.demo.yaml']


def run(*args, **kwargs):
    return subprocess.run(args, cwd=PLUGIN, check=True, **kwargs)


def main():
    action = sys.argv[1] if len(sys.argv) > 1 else 'start'
    if action in ('stop', 'logs'):
        if STATE.exists():
            run(*COMPOSE, *(['down'] if action == 'stop' else ['logs', '--tail=100', '-f']))
        return
    if action != 'start':
        raise SystemExit('Usage: run.py [start|stop|logs]')
    STATE.mkdir(mode=0o700, exist_ok=True)
    arch = run('docker', 'version', '--format', '{{.Server.Arch}}', capture_output=True, text=True).stdout.strip()
    build_env = {**os.environ, 'GOOS': 'linux', 'GOARCH': arch, 'CGO_ENABLED': '0'}
    subprocess.run(['go', 'build', '-o', str(STATE / 'apiserver'), './cmd/apiserver'], cwd=ROOT, env=build_env, check=True)
    run(*COMPOSE, 'up', '-d', '--build', '--force-recreate', 'commander', 'collector-a', 'collector-b')
    address = 'http://127.0.0.1:' + os.environ.get('COMMANDER_DEMO_PORT', '3301')
    request = address + '/api/v1/namespaces/default/agents?limit=100'
    for _ in range(60):
        try:
            with urllib.request.urlopen(request, timeout=5) as response:
                agents = json.load(response)['items']
        except (urllib.error.URLError, TimeoutError):
            time.sleep(1)
            continue
        if len([agent for agent in agents if agent['status']['connected']]) >= 2:
            break
        time.sleep(1)
    else:
        raise SystemExit('Demo collectors did not connect. Run make grafana-demo-logs.')
    # Keep a real disconnected record visible, while reducing the demo's memory use.
    run(*COMPOSE, 'stop', 'collector-b')
    run(*COMPOSE, 'up', '-d', '--wait', '--force-recreate', '--no-deps', 'grafana')
    port = os.environ.get('GRAFANA_PORT', '3300')
    print(f'Fleet demo: http://127.0.0.1:{port}/a/minukdev-opampcommander-app', flush=True)
    print('Two real collectors connect to the private demo backend.', flush=True)


if __name__ == '__main__':
    main()
