oc delete mutatingwebhookconfiguration.admissionregistration.k8s.io/inject-logstach-api-logger-sidecar

oc delete service petclinic -n myproject
oc delete deployment petclinic -n myproject

sleep 3
oc apply -f yaml/MutatingWebhookConfiguration.yaml