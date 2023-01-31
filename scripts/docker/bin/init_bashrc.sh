alias h='cd /usr/local/holoinsight/agent'
alias g='cd /usr/local/holoinsight/agent/logs'

if [ "$HI_AGENT_MODE" = "daemonset" ]; then
  alias tohost='chroot /$HOSTFS bash'
  alias tohostns='nsenter -t 1 -m -u -i -n -F bash --restricted -c bash'
fi
