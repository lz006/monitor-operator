# Monitor Operator

This project aims to automate "prometheus operator monitoring" further.
Goals are:
 - Integration of targets that are located outside of kubernetes/openshift
 - Automated creation of ServiceMonitor, Service & Endpoints resource
 - Mapping from awx/ansible-tower inventory to prometheus crds through this new "Monitor Operator"

 ## Project is in a very early state and is not intended for production use