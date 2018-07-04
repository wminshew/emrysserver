FROM golang:1.9.2-alpine
COPY . /go/src/github.com/wminshew/emrysserver
RUN go install github.com/wminshew/emrysserver

FROM alpine:latest
COPY --from=0 /go/bin/emrysserver .
CMD ["./emrysserver"]
