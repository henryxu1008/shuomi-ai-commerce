#!/bin/sh
set -eu
umask 077
test -s /base/current-application.json
test -s /secrets/acquisition-owner/acquisition-db-password
jq -e '.productCollections == true and .productAcquisitionDatabase.database == "product_acquisition" and .productAcquisitionDatabase.user == "source_acquisition_runtime"' /base/current-application.json >/dev/null
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
jq -n --rawfile password /secrets/acquisition-owner/acquisition-db-password \
 '{schemaVersion:1,collections:true,acquisitionPlans:true,database:{host:"127.0.0.1",port:5433,user:"acquisition_owner",password:($password|rtrimstr("\n")),database:"product_acquisition"}}' > "$work/product.json"
product-acquisition-init -config "$work/product.json" -enable-acquisition-plans -confirm-database product_acquisition
if test ! -s /collector-private/credential; then
 openssl rand -hex 32 | tr -d '\n' > /collector-private/credential
 chgrp 65534 /collector-private/credential
 chmod 440 /collector-private/credential
fi
jq --rawfile credential /collector-private/credential \
 '.acquisitionPlans=true | .browserCollector={endpoint:"http://browser-collector:19545",credential:$credential,timeout:40000000000}' \
 /base/current-application.json > /runtime/current-application.json.tmp
chmod 600 /runtime/current-application.json.tmp
mv /runtime/current-application.json.tmp /runtime/current-application.json
echo 'Acquisition plans configured; existing Product facts and grants retained'
