#!/bin/bash

version=0.0.4
docker build -t basilnsage/mwn-ticketapp.crud:"$version" -t basilnsage/mwn-ticketapp.crud:latest .
