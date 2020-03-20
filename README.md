# Crypto to Fiat Forex Rates node
An external node implementation for Proxeus core to convert various crypto and fiat currencies at current market rates

## Implementation

Current implementation supports data retrieval from cryptocompare.com

## Usage

It is recommended to start it using docker.

The latest image is available at `proxeus/node-crypto-forex-rates:latest`

See the configuration paragraph for more information on what environments variables can be overridden

## Configuration

The following parameters can be set via environment variables. 


| Environmentvariable | Default value
--- | --- |  
PROXEUS_INSTANCE_URL | http://127.0.0.1:1323
SERVICE_NAME | Crypto to Fiat Forex Rates
SERVICE_URL | http://localhost:SERVICE_PORT
SERVICE_PORT | 8011
SERVICE_SECRET | my secret
REGISTER_RETRY_INTERVAL | 5

## Deployment

The node is available as docker image and can be used within a typical Proxeus Platform setup by including the following docker-compose service:

```
version: '3.7'

networks:
  xes-platform-network:
    name: xes-platform-network

services:
  node-crypto-forex-rates:
    image: proxeus/node-crypto-forex-rates:latest
    container_name: xes_node-crypto-forex-rates
    networks:
      - xes-platform-network
    restart: unless-stopped
    environment:
      PROXEUS_INSTANCE_URL: http://xes-platform:1323
      SERVICE_SECRET: secret
      SERVICE_PORT: 8011
      REGISTER_RETRY_INTERVAL: 3
      SERVICE_URL: http://node-crypto-forex-rates:8011
      TZ: Europe/Zurich
    ports:
      - "8011:8011"
```
