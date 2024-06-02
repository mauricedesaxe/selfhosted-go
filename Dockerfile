FROM golang:1.21.3-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN apk add --no-cache make curl gcc musl-dev

ARG ARCH=linux-x64
ARG ENV=prod

RUN make setup ARCH=${ARCH}
RUN make build ENV=${ENV}
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o bin/app .

FROM alpine:latest  

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/bin/app /root/app
COPY --from=builder /app/db /root/db
COPY --from=builder /app/public /root/public

EXPOSE 3000

CMD ["/root/app"]