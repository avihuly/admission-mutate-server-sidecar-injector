oc config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}'

oc config view --raw --minify --flatten -o yaml