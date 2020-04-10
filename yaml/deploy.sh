oc delete mutatingwebhookconfiguration.admissionregistration.k8s.io/inject-logstach-api-logger-sidecar
oc delete secret ssl

oc delete service petclinic -n myproject
oc delete deployment petclinic -n myproject

sleep 3
oc create secret generic ssl --from-file mutateme-server.pem=./ssl/mutateme-server.pem --from-file mutateme-server.key=./ssl/mutateme-server.key
oc apply -f yaml/MutatingWebhookConfiguration.yaml