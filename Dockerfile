# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot

# For multi-arch builds, use automatic platform build arguments
# see https://docs.docker.com/engine/reference/builder/#automatic-platform-args-in-the-global-scope
ARG TARGETOS
ARG TARGETARCH

WORKDIR /
USER 65532:65532
ADD "bin/manager_${TARGETOS}_${TARGETARCH}" "/manager"
ENTRYPOINT ["/manager"]
