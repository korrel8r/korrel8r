#!/bin/bash
# For the record: these commands were used to initialize the project.

set -ex

operator-sdk init --domain openshift.io --owner "Alan Conway" --project-name korrel8r
operator-sdk create api --group korrel8r --version v1alpha1 --kind Korrel8r --resource --controller
make
