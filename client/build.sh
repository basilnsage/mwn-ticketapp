#!/bin/bash

if [ -z "$1" ]; then
  echo "please specify a build number"
  exit 1
fi

docker build -t basilnsage/mwn-ticketapp.client:latest -t basilnsage/mwn-ticketapp.client:"$1" .
