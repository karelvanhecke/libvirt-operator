FROM docker.io/golang:1.24.1-bookworm@sha256:d7d795d0a9f51b00d9c9bfd17388c2c626004a50c6ed7c581e095122507fe1ab AS build
ARG VERSION
WORKDIR /usr/src/github.com/karelvanhecke/libvirt-operator

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make build-binary

FROM gcr.io/distroless/static-debian12:nonroot@sha256:c0f429e16b13e583da7e5a6ec20dd656d325d88e6819cafe0adb0828976529dc
COPY --from=build /usr/src/github.com/karelvanhecke/libvirt-operator/bin/operator /bin/operator
ENTRYPOINT [ "operator" ]
