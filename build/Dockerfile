FROM quay.io/app-sre/boilerplate:image-v2.1.0 AS builder

ENV OPERATOR=/usr/local/bin/ocm-agent-operator \
    USER_UID=1001 \
    USER_NAME=ocm-agent-operator

# install operator binary
COPY build/_output/bin/ocm-agent-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
