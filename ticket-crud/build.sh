#!/bin/bash

if [ -z "$1" ]
then
  echo "Please specify the version"
  exit 1
fi

docker build -t basilnsage/mwn-ticketapp.crud:"$1" -t basilnsage/mwn-ticketapp.crud:latest .
