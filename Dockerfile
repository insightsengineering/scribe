FROM scratch
ARG PROJECT_NAME="scribe"
# Uncomment when goreleaser config is updated to dockers_v2.
# ARG TARGETPLATFORM
ENV PROJECT_NAME=${PROJECT_NAME}
COPY ${PROJECT_NAME} /
# Change to this when goreleaser config is updated to dockers_v2.
# COPY $TARGETPLATFORM/${PROJECT_NAME} /
WORKDIR /
# hadolint ignore=DL3025
ENTRYPOINT ${PROJECT_NAME}
