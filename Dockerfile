FROM docker.io/golang:1.23.4-bookworm@sha256:6546e4d3271d1ba19159f10b77c04afa524afd0d1acc52cdae51e8a9e0399149 AS build
ARG OPERATOR_VERSION
WORKDIR /usr/src/github.com/karelvanhecke/libvirt-operator

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make build-binary

FROM gcr.io/distroless/static-debian12:nonroot@sha256:6cd937e9155bdfd805d1b94e037f9d6a899603306030936a3b11680af0c2ed58
COPY --from=build /usr/src/github.com/karelvanhecke/libvirt-operator/bin/operator /bin/operator
ENTRYPOINT [ "operator" ]
