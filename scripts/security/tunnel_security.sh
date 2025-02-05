#!/bin/bash

# Sprawdzenie czy skrypt jest uruchomiony z uprawnieniami roota
if [ "$EUID" -ne 0 ]; then 
    echo "Ten skrypt wymaga uprawnień roota"
    exit 1
fi

# Funkcja konfigurująca zabezpieczenia dla pojedynczego tunelu
configure_tunnel_security() {
    local IFACE=$1
    
    # Ograniczenie prędkości do 50 Mbps
    tc qdisc add dev $IFACE root handle 1: htb default 10
    tc class add dev $IFACE parent 1: classid 1:1 htb rate 50mbit ceil 50mbit
    tc filter add dev $IFACE parent 1: protocol ipv6 prio 1 u32 match ip6 src ::/0 flowid 1:1
    
    # Blokowanie portów email
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --dports 25,465,587,2525 -j DROP
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --sports 25,465,587,2525 -j DROP
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --dports 110,143,993,995 -j DROP
    iptables -A FORWARD -i $IFACE -p tcp -m multiport --sports 110,143,993,995 -j DROP
    
    # Ochrona przed DDoS
    iptables -A FORWARD -i $IFACE -p tcp --syn -m limit --limit 1/s -j ACCEPT
    iptables -A FORWARD -i $IFACE -p tcp --syn -j DROP
    iptables -A FORWARD -i $IFACE -p icmpv6 -m limit --limit 1/s -j ACCEPT
    iptables -A FORWARD -i $IFACE -p icmpv6 -j DROP
    iptables -A FORWARD -i $IFACE -m hashlimit --hashlimit-above 10000/sec --hashlimit-burst 100 --hashlimit-mode srcip --hashlimit-name ${IFACE}_ddos -j DROP
    
    # Dodatkowe zabezpieczenia
    iptables -A FORWARD -i $IFACE -f -j DROP
    iptables -A FORWARD -i $IFACE -p tcp --tcp-flags ALL NONE -j DROP
    iptables -A FORWARD -i $IFACE -p tcp --tcp-flags ALL ALL -j DROP
    iptables -A FORWARD -i $IFACE -m state --state NEW -m limit --limit 50/second --limit-burst 50 -j ACCEPT
    
    echo "Skonfigurowano zabezpieczenia dla interfejsu $IFACE"
}

# Konfiguracja dla wszystkich istniejących tuneli
for iface in $(ip link show | grep 'tun-' | cut -d: -f2 | tr -d ' '); do
    configure_tunnel_security $iface
done

# Zapisanie reguł iptables
iptables-save > /etc/iptables/rules.v4
ip6tables-save > /etc/iptables/rules.v6

echo "Zakończono konfigurację zabezpieczeń dla wszystkich tuneli" 