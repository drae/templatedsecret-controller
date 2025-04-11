FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY templated-secret-controller /templated-secret-controller

# Use nonroot user from distroless image
USER 65532:65532

ENTRYPOINT ["/templated-secret-controller"]