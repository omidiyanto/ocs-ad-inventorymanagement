#!/bin/bash
docker build -t registry.satnusa.com/ocs-ad-inventory-management:latest .
# docker push registry.satnusa.com/ocs-ad-inventory-management:latest
docker-compose -f docker-compose.yaml down ocs-ad-inventory
docker-compose -f docker-compose.yaml up -d --build ocs-ad-inventory