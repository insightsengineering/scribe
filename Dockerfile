FROM scratch
ARG PROJECT_NAME="scribe"
ENV PROJECT_NAME=${PROJECT_NAME}
COPY ${PROJECT_NAME} /
WORKDIR /
# hadolint ignore=DL3025
ENTRYPOINT ${PROJECT_NAME}
