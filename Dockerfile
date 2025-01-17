FROM docker.io/golang:1.23.5-bookworm@sha256:3149bc5043fa58cf127fd8db1fdd4e533b6aed5a40d663d4f4ae43d20386665f AS build
ARG VERSION
WORKDIR /usr/src/github.com/karelvanhecke/libvirt-operator

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN make build-binary

FROM gcr.io/distroless/static-debian12:nonroot@sha256:6ec5aa99dc335666e79dc64e4a6c8b89c33a543a1967f20d360922a80dd21f02
COPY --from=build /usr/src/github.com/karelvanhecke/libvirt-operator/bin/operator /bin/operator
ENTRYPOINT [ "operator" ]
