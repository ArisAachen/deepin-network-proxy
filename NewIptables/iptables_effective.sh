#!/bin/sh

## main
start_main() {
  ## create main chain to filter all proc, rules except first chain
  iptables -t mangle -N All_Entry
  iptables -t mangle -A OUTPUT -j All_Entry

  ## create default chain, dont proxy main-proc, highest priority　in all chain
  start_default

  ## must not
  iptables -t mangle -A All_Entry -m cgroup --path main.slice -j RETURN
}

## never entry AllChain rules, including dns and other chain rules
start_default() {
  ## start default
  iptables -t mangle -N No_Proxy
  ## attach to main chain
  iptables -t mangle -A All_Entry -j No_Proxy
  ## add default no proxy, should use return as default
}

## create app proxy chain, use to app proxy
start_app() {
  iptables -t mangle -N App_Proxy
  iptables -t mangle -I All_Entry $1 -p tcp -m cgroup --path app.slice -j App_Proxy
  iptables -t mangle -A App_Proxy -j MARK --set-mark $2
}

## create global proxy chain, use to global proxy
start_global() {
  iptables -t mangle -N Global_Proxy
  iptables -t mangle -I All_Entry $1 -p tcp -m cgroup !--path global.slice -j Global_Proxy
  iptables -t mangle -A Global_Proxy -j MARK --set-mark $2
}

## stop app proxy
stop_app() {
  ## clear main app chain
  iptables -t mangle -F App_Proxy
  ## detach app chain from all chain
  iptables -t mangle -D All_Entry -p tcp -m cgroup --path app.slice -j App_Proxy
  ## delete chain
  iptables -t mangle -X App_Proxy
}

# stop global proxy
stop_global() {
  ## clear main app chain
  iptables -t mangle -F Global_Proxy
  ## detach app chain from all chain
  iptables -t mangle -D All_Entry -p tcp -m cgroup !--path global.slice -j Global_Proxy
  ## delete chain
  iptables -t mangle -X Global_Proxy
}

## stop all
stop_all() {
  iptables -t mangle -D OUTPUT -j All_Entry
  iptables -t mangle -X All_Entry
}
