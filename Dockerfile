FROM --platform=$BUILDPLATFORM golang:1.24.1 AS build

ARG TARGETOS TARGETARCH SGCTRL_VER=development
WORKDIR /workspace

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -mod=vendor -ldflags="-X 'main.Version=$SGCTRL_VER' -buildid=" -trimpath \
    -o templatedsecret-controller ./cmd/controller/...

# Use distroless as minimal base image to package the controller binary
FROM gcr.io/distroless/static:nonroot AS runtime

WORKDIR /
COPY --from=build /workspace/templatedsecret-controller /templatedsecret-controller

# Use nonroot user from distroless image
USER 65532:65532

ENTRYPOINT ["/templatedsecret-controller"]
