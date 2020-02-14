FROM alpine
ARG BIN_NAME=node-crypto-forex-rates
WORKDIR /app
COPY /artifacts/$BIN_NAME /app/node
EXPOSE 8011
ENTRYPOINT ["./node"]
