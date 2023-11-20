ARG BASE_IMAGE
FROM ${BASE_IMAGE}

COPY LICENSE /licenses/
COPY LICENSE.3RD_PARTY /licenses/
COPY forwarder /usr/bin
ENTRYPOINT ["/usr/bin/forwarder"]
CMD ["run"]
