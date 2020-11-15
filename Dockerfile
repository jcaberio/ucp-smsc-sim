FROM golang:1.11 AS build
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o /out/ucp-smsc-sim .

FROM scratch AS bin
COPY --from=build /out/ucp-smsc-sim /
ENTRYPOINT ["./ucp-smsc-sim"]
EXPOSE 16003
