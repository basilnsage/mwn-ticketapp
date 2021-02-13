#!/bin/bash

version="0.0.4"
repo=basilnsage
image=mwn-ticketapp.auth

docker build -t "$repo/$image:latest" -t "$repo/$image:$version" .
