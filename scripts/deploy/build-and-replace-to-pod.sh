#!/usr/bin/env bash
set -e

cd `dirname $0`/../..

ns=holoinsight-agent
context=$1
pod=$2

echo context is $context
echo pod is $pod
echo

if [ -z "$context" ] || [ -z "$pod" ]; then
  echo 'usage: <build-and-replace-to-pod.sh <context> <pod>'
  exit 1
fi

echo Build agent ...
./scripts/build/build-using-go.sh >/dev/null
echo Buil done
echo

echo Stop agent ...
kubectl --context $context -n $ns exec -i $pod -- bash -s <<EOF
  sc stop app
EOF
echo

echo Copy binaries to agent ...
kubectl --context $context -n $ns cp ./build/linux-amd64/bin/agent $pod:/usr/local/holoinsight/agent/bin/agent
kubectl --context $context -n $ns cp ./build/linux-amd64/bin/helper $pod:/usr/local/holoinsight/agent/bin/helper
echo Copy done
echo

echo Start agent ...
kubectl --context $context -n $ns exec -i $pod -- bash -s <<EOF
  sc start app
  echo Remote md5
  md5sum /usr/local/holoinsight/agent/bin/{agent,helper}
EOF
echo

echo Local md5
md5sum ./build/linux-amd64/bin/{agent,helper}
echo
