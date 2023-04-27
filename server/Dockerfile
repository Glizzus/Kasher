FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main .

FROM alpine
COPY --from=builder /app/main /app/main

ENV port=8080

EXPOSE ${port}

CMD /app/main ${port}