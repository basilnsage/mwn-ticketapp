#!/bin/bash

# pull verion from package.json
version="$(jq '.version' package.json | sed 's/"//g')"
docker build -t basilnsage/mwn-ticketapp.client:latest -t basilnsage/mwn-ticketapp.client:"$version" .
