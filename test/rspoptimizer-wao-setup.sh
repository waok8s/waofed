#!/usr/bin/env bash

# scripts must be run from project root
. hack/2-lib.sh || exit 1

# consts

# main

cluster0=$PROJECT_NAME-test-0
cluster1=$PROJECT_NAME-test-1

lib::install-wao-estimator "$cluster0" 
lib::install-wao-estimator "$cluster1"

sleep 30

lib::start-wao-estimator "$cluster0" "./test/rspoptimizer-wao-estimator.yaml" "5657"
lib::start-wao-estimator "$cluster1" "./test/rspoptimizer-wao-estimator.yaml" "5658"
