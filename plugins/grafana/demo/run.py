#!/usr/bin/env python3
"""Run a disposable, authenticated local Commander + real collector demo."""
import base64
import json
import os
from pathlib import Path
import secrets
import shutil
import subprocess
import sys
import time
import urllib.error
import urllib.request

PLUGIN = Path(__file__).resolve().parents[1]
ROOT = PLUGIN.parents[1]
STATE = PLUGIN / '.demo'
ENV = STATE / 'demo.env'
COMPOSE = ['docker', 'compose', '--env-file', str(ENV), '-f', 'docker-compose.yaml', '-f', 'compose.demo.yaml']


def run(*args, **kwargs):
    return subprocess.run(args, cwd=PLUGIN, check=True, **kwargs)


def write_environment(values):
    ENV.touch(mode=0o600)
    ENV.chmod(0o600)
    ENV.write_text(''.join(f'{key}={value}\n' for key, value in values.items()))


def main():
    action = sys.argv[1] if len(sys.argv) > 1 else 'start'
    if action in ('stop', 'logs'):
        if ENV.exists():
            run(*COMPOSE, *(['down'] if action == 'stop' else ['logs', '--tail=100', '-f']))
        return
    if action != 'start':
        raise SystemExit('Usage: run.py [start|stop|logs]')
    STATE.mkdir(mode=0o700, exist_ok=True)
    initial = STATE / 'initial'
    initial.mkdir(exist_ok=True)
    for path in (ROOT / 'configs/apiserver/initial').glob('*.yaml'):
        shutil.copyfile(path, initial / path.name)
    password = secrets.token_hex(24)
    (initial / '20-grafana-demo.yaml').write_text(json.dumps({
        'apiVersion': 'v1', 'kind': 'User',
        'spec': {'username': 'grafana-demo', 'email': 'grafana-demo@localhost', 'isActive': True, 'password': password},
    }))
    values = {key: secrets.token_hex(32) for key in ('COMMANDER_ADMIN_PASSWORD', 'COMMANDER_PEPPER', 'COMMANDER_JWT_SECRET')}
    values['COMMANDER_DEMO_TOKEN'] = ''
    write_environment(values)
    arch = run('docker', 'version', '--format', '{{.Server.Arch}}', capture_output=True, text=True).stdout.strip()
    build_env = {**os.environ, 'GOOS': 'linux', 'GOARCH': arch, 'CGO_ENABLED': '0'}
    subprocess.run(['go', 'build', '-o', str(STATE / 'apiserver'), './cmd/apiserver'], cwd=ROOT, env=build_env, check=True)
    run(*COMPOSE, 'up', '-d', '--build', '--force-recreate', 'commander', 'collector-a', 'collector-b')
    address = 'http://127.0.0.1:' + os.environ.get('COMMANDER_DEMO_PORT', '3301')
    credentials = base64.b64encode(f'grafana-demo:{password}'.encode()).decode()
    request = urllib.request.Request(address + '/api/v1/auth/basic', headers={'Authorization': 'Basic ' + credentials})
    token = None
    for _ in range(60):
        try:
            with urllib.request.urlopen(request, timeout=2) as response:
                token = json.load(response)['token']
            break
        except (urllib.error.URLError, TimeoutError):
            time.sleep(1)
    if not token:
        raise SystemExit('Commander login did not become ready. Run make grafana-demo-logs.')
    values['COMMANDER_DEMO_TOKEN'] = token
    write_environment(values)
    request = urllib.request.Request(address + '/api/v1/namespaces/default/agents?limit=100', headers={'Authorization': 'Bearer ' + token})
    for _ in range(60):
        with urllib.request.urlopen(request, timeout=5) as response:
            agents = json.load(response)['items']
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
    print('Two real demo collectors; read-only token expires in 24h. Restart the demo to renew it.', flush=True)


if __name__ == '__main__':
    main()
