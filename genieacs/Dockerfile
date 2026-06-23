# GenieACS TR-069 (CWMP) engine — runs cwmp / nbi / fs as separate compose services
# from this one image. The backend app proxies /cwmp -> genieacs-cwmp and talks to
# genieacs-nbi (REST). MongoDB is the datastore.
FROM node:18-bookworm-slim

RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl \
 && rm -rf /var/lib/apt/lists/* \
 && npm install -g genieacs@1.2.13 \
 && npm cache clean --force

# Extension dir (for provision-called extensions; harmless if empty)
RUN useradd -m genieacs \
 && mkdir -p /opt/genieacs/ext \
 && chown -R genieacs:genieacs /opt/genieacs

USER genieacs
ENV GENIEACS_EXT_DIR=/opt/genieacs/ext

# Default; overridden per-service in docker-compose
CMD ["genieacs-cwmp"]
