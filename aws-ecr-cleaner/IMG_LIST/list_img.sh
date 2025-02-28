#!/bin/bash

{
  # Pods（包括 containers、initContainers 和 ephemeralContainers）
  kubectl get pods --all-namespaces -o jsonpath="{range .items[*]}{range .spec.containers[*]}{.image}{'\n'}{end}{range .spec.initContainers[*]}{.image}{'\n'}{end}{range .spec.ephemeralContainers[*]}{.image}{'\n'}{end}{end}"
  
  # Deployments
  kubectl get deploy --all-namespaces -o jsonpath="{range .items[*]}{range .spec.template.spec.containers[*]}{.image}{'\n'}{end}{range .spec.template.spec.initContainers[*]}{.image}{'\n'}{end}{end}"
  
  # StatefulSets
  kubectl get sts --all-namespaces -o jsonpath="{range .items[*]}{range .spec.template.spec.containers[*]}{.image}{'\n'}{end}{range .spec.template.spec.initContainers[*]}{.image}{'\n'}{end}{end}"
  
  # DaemonSets
  kubectl get ds --all-namespaces -o jsonpath="{range .items[*]}{range .spec.template.spec.containers[*]}{.image}{'\n'}{end}{range .spec.template.spec.initContainers[*]}{.image}{'\n'}{end}{end}"
  
  # Jobs
  kubectl get job --all-namespaces -o jsonpath="{range .items[*]}{range .spec.template.spec.containers[*]}{.image}{'\n'}{end}{range .spec.template.spec.initContainers[*]}{.image}{'\n'}{end}{end}"
  
  # CronJobs（注意 CronJob 的 pod 模板在 .spec.jobTemplate.spec.template 内）
  kubectl get cronjob --all-namespaces -o jsonpath="{range .items[*]}{range .spec.jobTemplate.spec.template.spec.containers[*]}{.image}{'\n'}{end}{range .spec.jobTemplate.spec.template.spec.initContainers[*]}{.image}{'\n'}{end}{end}"
  
} | sort -n | uniq -c | awk '{print $2}' | cut -d'/' -f2-


