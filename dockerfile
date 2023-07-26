FROM golang AS build-env
COPY . /go-src/discord
WORKDIR /go-src/discord
RUN go test -v /go-src/discord
RUN go build -o /go-app .

FROM gcr.io/distroless/base
COPY --from=build-env /go-app /
ENTRYPOINT ["/go-app", "--alsologtostderr", "--v=0"]
