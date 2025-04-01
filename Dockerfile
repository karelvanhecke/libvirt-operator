FROM docker.io/golang:1.24.2-bookworm@sha256:75e6700eab3c994f730e36f357a26ee496b618d51eaecb04716144e861ad74f3 AS build
ARG VERSION
WORKDIR /usr/src/github.com/karelvanhecke/libvirt-operator

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make build-binary

FROM gcr.io/distroless/static-debian12:nonroot@sha256:6ec5aa99dc335666e79dc64e4a6c8b89c33a543a1967f20d360922a80dd21f02
COPY --from=build /usr/src/github.com/karelvanhecke/libvirt-operator/bin/operator /bin/operator
ENTRYPOINT [ "operator" ]
