#!/bin/bash

export BASE64PKEY=$(base64 ../private.pem)
export APIKEY=$(cat ../apikey_$1)
