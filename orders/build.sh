#!/bin/bash

version=0.0.1
docker build -t  basilnsage/mwn-ticketapp.orders:"$version" -t basilnsage/mwn-ticketapp.orders:latest .
