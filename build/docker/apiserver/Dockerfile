FROM alpine:3.21
# Absolute path: the binary is copied to / , which is not on PATH, so a bare
# "apiserver" entrypoint fails with "executable file not found in $PATH".
ENTRYPOINT [ "/apiserver" ]
COPY apiserver /
# Default initial manifests, reconciled into persistence on startup. They are kept
# as plain files (not embedded) so operators can inspect them in the image, edit
# them, or point bootstrap.dir / BOOTSTRAP_DIR elsewhere.
COPY configs/apiserver/initial /etc/opampcommander/initial
ENV BOOTSTRAP_DIR=/etc/opampcommander/initial
