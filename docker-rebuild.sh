#!/bin/bash
docker build -t registry.satnusa.com/ocs-ad-inventory-management:latest .
docker push registry.satnusa.com/ocs-ad-inventory-management:latest
docker-compose -f docker-compose.yml down
docker-compose -f docker-compose.yml up -d --build