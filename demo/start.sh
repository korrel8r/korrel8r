
# Crash-looper to get alert, takes 5m
oc delete deployment demo
oc create deployment demo --image=quay.io/quay/busybox -- sh -c 'for i in range 6 do; echo $(date) here we go $i; sleep 1; echo $(date) Oh dear, oh dear; exit 1'
