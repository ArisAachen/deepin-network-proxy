#!/bin/bash

## clear app iptables
clear_app_iptables(){
    ## clear app chain
    iptables -t mangle -F App
    ## detach app chain from main
    iptables -t mangle -D Main -j App -p tcp -m cgroup --path App.slice
    ## remove chain
    iptables -t mangle -X App

    ## del mark rule from output chain
    iptables -t mangle -D PREROUTING -j TPROXY -p tcp --on-port 8090 -m mark --mark 8090
}

## clear app ip rule
clear_app_iprule(){
    ## delete rule
    ip rule del fwmark 8090 table 100
}

## clear app proxy setting
clear_app(){
    clear_app_iptables
    clear_app_iprule
}

## clear global iptables
clear_global_iptables(){
   ## clear global chain
    iptables -t mangle -F Global
    ## detach global chain from main
    iptables -t mangle -D Main -j Global -p tcp -m cgroup --path Global.slice
    ## remove chain
    iptables -t mangle -X Global

    ## del mark rule from output chain
    iptables -t mangle -D PREROUTING -j TPROXY -p tcp --on-port 8080 -m mark --mark 8080 
}

## clear global ip rule
clear_global_iprule(){
    ## delete rule
    ip rule del fwmark 8080 table 100
}

## clear global proxy setting
clear_global(){
    clear_global_iptables
    clear_global_iprule
}

## clear main iptables
clear_main_iptables(){
    ## clear main rules
    iptables -t mangle -F Main
    ## detach main rule from OUTPUT chain
    iptables -t mangle -D OUTPUT -j Main
    ## remove main chain
    iptables -t mangle -X Main
}

## clear main ip route
clear_main_route(){
    ## remove ip route
    ip route del local default dev lo table 100
}

## clear main 
clear_main(){
    clear_main_iptables
    clear_main_route
}

echo "begin clear" + $1


for i in "$@"
do
case $i in
    clear_Main)
        clear_main
        exit 0
        ;;
    clear_Global)
        clear_global
        exit 0
        ;;
    clear_App)
        clear_app
        exit 0
        ;;
    *)
        exit 0
        ;;
esac
done