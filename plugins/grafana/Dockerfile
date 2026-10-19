# Build dist first with `make build`. Both Linux architectures are included.
ARG GRAFANA_VERSION=13.2.4
FROM grafana/grafana:${GRAFANA_VERSION}

# /var/lib/grafana is mounted by Helm; baking plugins there would hide them.
COPY --chown=472:0 dist/ /opt/opampcommander/plugins/minukdev-opampcommander-app/
ENV GF_PATHS_PLUGINS=/opt/opampcommander/plugins
