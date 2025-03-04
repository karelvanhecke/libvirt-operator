FROM docker.io/golang:1.24.1-bookworm@sha256:eb37f58646a901dc7727cf448cae36daaefaba79de33b5058dab79aa4c04aefb AS build
ARG VERSION
WORKDIR /usr/src/github.com/karelvanhecke/libvirt-operator

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make build-binary

FROM gcr.io/distroless/static-debian12:nonroot@sha256:6ec5aa99dc335666e79dc64e4a6c8b89c33a543a1967f20d360922a80dd21f02
COPY --from=build /usr/src/github.com/karelvanhecke/libvirt-operator/bin/operator /bin/operator
ENTRYPOINT [ "operator" ]
