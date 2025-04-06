#!/bin/bash

# Sprawdzenie czy skrypt jest uruchomiony z uprawnieniami roota
if [ "$EUID" -ne 0 ]; then 
    echo "Ten skrypt wymaga uprawnień roota"
    exit 1
fi

# Funkcja konfigurująca zabezpieczenia dla pojedynczego tunelu
configure_tunnel_security() {
    local IFACE=$(echo "$1" | cut -d@ -f1)
    
    # Usuwanie istniejących konfiguracji
    tc qdisc del dev $IFACE root 2>/dev/null
    tc qdisc del dev $IFACE ingress 2>/dev/null
    
    # Konfiguracja dla ruchu wychodzącego (egress/upload)
    tc qdisc add dev $IFACE root handle 1: htb default 10
    tc class add dev $IFACE parent 1: classid 1:1 htb rate 50mbit ceil 50mbit
    tc filter add dev $IFACE parent 1: protocol ipv6 prio 1 u32 match ip6 src ::/0 flowid 1:1
    
    # Konfiguracja dla ruchu przychodzącego (ingress/download)
    tc qdisc add dev $IFACE handle ffff: ingress
    tc filter add dev $IFACE parent ffff: protocol ipv6 u32 match u32 0 0 police rate 50mbit burst 50k drop flowid :1
    
    # IPv4 rules (dla ruchu tunelowanego)
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --dports 25,465,587,2525 -j DROP
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --sports 25,465,587,2525 -j DROP
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --dports 110,143,993,995 -j DROP
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --sports 110,143,993,995 -j DROP
    
    # Ochrona przed DDoS (IPv4)
    iptables -A FORWARD -i $IFACE -p tcp --syn -m limit --limit 1/s -j ACCEPT
    iptables -A FORWARD -i $IFACE -p tcp --syn -j DROP
    iptables -A FORWARD -i $IFACE -p icmp -m limit --limit 1/s -j ACCEPT
    iptables -A FORWARD -i $IFACE -p icmp -j DROP
    iptables -A FORWARD -i $IFACE -m hashlimit --hashlimit-above 10000/sec --hashlimit-burst 100 --hashlimit-mode srcip --hashlimit-name ${IFACE}_ddos -j DROP
    
    # Dodatkowe zabezpieczenia IPv4
    iptables -A FORWARD -i $IFACE -f -j DROP
    iptables -A FORWARD -i $IFACE -p tcp --tcp-flags ALL NONE -j DROP
    iptables -A FORWARD -i $IFACE -p tcp --tcp-flags ALL ALL -j DROP
    iptables -A FORWARD -i $IFACE -m state --state NEW -m limit --limit 50/second --limit-burst 50 -j ACCEPT

    # IPv6 rules (dodatkowa warstwa bezpieczeństwa)
    ip6tables -A FORWARD -i $IFACE -p tcp -m multiport --dports 25,465,587,2525 -j DROP
    ip6tables -A FORWARD -i $IFACE -p tcp -m multiport --sports 25,465,587,2525 -j DROP
    ip6tables -A FORWARD -i $IFACE -p tcp -m multiport --dports 110,143,993,995 -j DROP
    ip6tables -A FORWARD -i $IFACE -p tcp -m multiport --sports 110,143,993,995 -j DROP
    
    # Ochrona przed DDoS (IPv6)
    ip6tables -A FORWARD -i $IFACE -p tcp --syn -m limit --limit 1/s -j ACCEPT
    ip6tables -A FORWARD -i $IFACE -p tcp --syn -j DROP
    ip6tables -A FORWARD -i $IFACE -p icmpv6 -m limit --limit 1/s -j ACCEPT
    ip6tables -A FORWARD -i $IFACE -p icmpv6 -j DROP
    ip6tables -A FORWARD -i $IFACE -m hashlimit --hashlimit-above 10000/sec --hashlimit-burst 100 --hashlimit-mode srcip --hashlimit-name ${IFACE}_ddos_v6 -j DROP
    
    # Dodatkowe zabezpieczenia IPv6
    ip6tables -A FORWARD -i $IFACE -m frag --fragfirst -j DROP
    ip6tables -A FORWARD -i $IFACE -p tcp --tcp-flags ALL NONE -j DROP
    ip6tables -A FORWARD -i $IFACE -p tcp --tcp-flags ALL ALL -j DROP
    ip6tables -A FORWARD -i $IFACE -m state --state NEW -m limit --limit 50/second --limit-burst 50 -j ACCEPT
    
    # Dodatkowe reguły specyficzne dla IPv6
    ip6tables -A FORWARD -i $IFACE -p icmpv6 --icmpv6-type redirect -j DROP
    ip6tables -A FORWARD -i $IFACE -p icmpv6 --icmpv6-type router-advertisement -j DROP
    ip6tables -A FORWARD -i $IFACE -p icmpv6 --icmpv6-type router-solicitation -j DROP
    
    echo "Skonfigurowano zabezpieczenia IPv4 i IPv6 dla interfejsu $IFACE"
}

# Usuwanie istniejących reguł
iptables -F FORWARD
ip6tables -F FORWARD
tc qdisc del dev tun+ root 2>/dev/null
tc qdisc del dev tun+ ingress 2>/dev/null

# Konfiguracja dla wszystkich istniejących tuneli
for iface in $(ip link show | grep '^[0-9]*: tun-' | cut -d: -f2 | tr -d ' ' | cut -d@ -f1); do
    configure_tunnel_security $iface
done

# Zapisanie reguł iptables
mkdir -p /etc/iptables
iptables-save > /etc/iptables/rules.v4
ip6tables-save > /etc/iptables/rules.v6

echo "Zakończono konfigurację zabezpieczeń dla wszystkich tuneli" 
