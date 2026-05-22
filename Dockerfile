FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o HLTV-Manager .

FROM ubuntu:latest
RUN apt-get update && apt-get install -y \
    curl bash docker.io \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/HLTV-Manager .
COPY frontend /app/frontend
RUN chmod +x ./HLTV-Manager
VOLUME /var/run/docker.sock:/var/run/docker.sock
USER root
CMD ["./HLTV-Manager"]