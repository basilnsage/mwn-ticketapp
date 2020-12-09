#!/bin/bash

if [ -z "$1" ]
then
  echo "Please specify a Docker image tag"
  exit 1
fi

repo=basilnsage
image=mwn-ticketapp.auth

docker build -t "$repo/$image:latest" -t "$repo/$image:$1" .
