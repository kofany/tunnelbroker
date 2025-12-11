#!/bin/bash

# Sprawdzenie czy skrypt jest uruchomiony z uprawnieniami roota
if [ "$EUID" -ne 0 ]; then
    echo "Ten skrypt wymaga uprawnień roota"
    exit 1
fi

echo "Konfiguracja zabezpieczeń tuneli (batch mode)..."

# ============================================
# SEKCJA 1: SIT/GRE tunele (iptables + tc)
# ============================================

# Pobierz listę wszystkich interfejsów tunelowych SIT/GRE
TUNNELS=$(ip link show | grep -oP '(?<=: )tun-[^@:]+' | sort -u)
TUNNEL_COUNT=$(echo "$TUNNELS" | grep -c . 2>/dev/null || echo 0)

echo "Znaleziono $TUNNEL_COUNT tuneli SIT/GRE"

# Usuń stare reguły iptables
iptables -F FORWARD 2>/dev/null
ip6tables -F FORWARD 2>/dev/null

if [ "$TUNNEL_COUNT" -gt 0 ]; then
    # Generuj reguły iptables batch
    IPT4_RULES="*filter
:FORWARD ACCEPT [0:0]
"

    IPT6_RULES="*filter
:FORWARD ACCEPT [0:0]
"

    for IFACE in $TUNNELS; do
        # IPv4 rules
        IPT4_RULES+="-A FORWARD -i $IFACE -p tcp -m multiport --dports 25,465,587,2525 -j DROP
-A FORWARD -i $IFACE -p tcp -m multiport --sports 25,465,587,2525 -j DROP
-A FORWARD -i $IFACE -p tcp -m multiport --dports 110,143,993,995 -j DROP
-A FORWARD -i $IFACE -p tcp -m multiport --sports 110,143,993,995 -j DROP
-A FORWARD -i $IFACE -p tcp --syn -m limit --limit 10/s --limit-burst 20 -j ACCEPT
-A FORWARD -i $IFACE -p tcp --syn -j DROP
-A FORWARD -i $IFACE -p icmp -m limit --limit 5/s -j ACCEPT
-A FORWARD -i $IFACE -p icmp -j DROP
-A FORWARD -i $IFACE -f -j DROP
"

        # IPv6 rules
        IPT6_RULES+="-A FORWARD -i $IFACE -p tcp -m multiport --dports 25,465,587,2525 -j DROP
-A FORWARD -i $IFACE -p tcp -m multiport --sports 25,465,587,2525 -j DROP
-A FORWARD -i $IFACE -p tcp -m multiport --dports 110,143,993,995 -j DROP
-A FORWARD -i $IFACE -p tcp -m multiport --sports 110,143,993,995 -j DROP
-A FORWARD -i $IFACE -p tcp --syn -m limit --limit 10/s --limit-burst 20 -j ACCEPT
-A FORWARD -i $IFACE -p tcp --syn -j DROP
-A FORWARD -i $IFACE -p icmpv6 -m limit --limit 5/s -j ACCEPT
-A FORWARD -i $IFACE -p icmpv6 -j DROP
-A FORWARD -i $IFACE -p icmpv6 --icmpv6-type redirect -j DROP
-A FORWARD -i $IFACE -p icmpv6 --icmpv6-type router-advertisement -j DROP
"
    done

    IPT4_RULES+="COMMIT
"
    IPT6_RULES+="COMMIT
"

    # Załaduj reguły batch (1 wywołanie zamiast setek)
    echo "$IPT4_RULES" | iptables-restore -n
    echo "$IPT6_RULES" | ip6tables-restore -n

    echo "Załadowano reguły iptables dla $TUNNEL_COUNT tuneli SIT/GRE"

    # Konfiguracja QoS (tc) - równolegle
    configure_tc() {
        local IFACE=$1
        tc qdisc del dev $IFACE root 2>/dev/null
        tc qdisc del dev $IFACE ingress 2>/dev/null
        tc qdisc add dev $IFACE root handle 1: htb default 10 2>/dev/null
        tc class add dev $IFACE parent 1: classid 1:1 htb rate 50mbit ceil 50mbit 2>/dev/null
        tc qdisc add dev $IFACE handle ffff: ingress 2>/dev/null
        tc filter add dev $IFACE parent ffff: protocol ipv6 u32 match u32 0 0 police rate 50mbit burst 50k drop flowid :1 2>/dev/null
    }

    # Uruchom tc równolegle (max 20 na raz)
    echo "Konfiguracja QoS dla SIT/GRE..."
    PARALLEL=0
    for IFACE in $TUNNELS; do
        configure_tc $IFACE &
        PARALLEL=$((PARALLEL + 1))
        if [ $PARALLEL -ge 20 ]; then
            wait
            PARALLEL=0
        fi
    done
    wait
fi

# ============================================
# SEKCJA 2: WireGuard (nftables per-peer)
# ============================================

# Sprawdź czy wg0 istnieje
if ip link show wg0 &>/dev/null; then
    echo "Konfiguracja nftables dla WireGuard..."

    # Pobierz prefiksy WG peerów z allowed-ips
    WG_PREFIXES=$(wg show wg0 allowed-ips 2>/dev/null | awk '{for(i=2;i<=NF;i++) print $i}' | grep -v '^$')
    WG_PREFIX_COUNT=$(echo "$WG_PREFIXES" | grep -c . 2>/dev/null || echo 0)

    echo "Znaleziono $WG_PREFIX_COUNT prefixów WireGuard"

    # Usuń stare reguły nftables dla WG (jeśli istnieją)
    nft delete table inet wg_security 2>/dev/null

    # Utwórz nową tabelę nftables dla WireGuard
    nft add table inet wg_security

    # Chain dla forward - firewall rules
    nft add chain inet wg_security forward { type filter hook forward priority 0 \; policy accept \; }

    # Blokada portów mail dla wg0
    nft add rule inet wg_security forward iifname "wg0" tcp dport { 25, 465, 587, 2525, 110, 143, 993, 995 } drop
    nft add rule inet wg_security forward iifname "wg0" tcp sport { 25, 465, 587, 2525, 110, 143, 993, 995 } drop

    # SYN flood protection
    nft add rule inet wg_security forward iifname "wg0" tcp flags syn limit rate 10/second burst 20 packets accept
    nft add rule inet wg_security forward iifname "wg0" tcp flags syn drop

    # ICMPv6 limit
    nft add rule inet wg_security forward iifname "wg0" icmpv6 type { redirect, nd-router-advert } drop
    nft add rule inet wg_security forward iifname "wg0" meta l4proto icmpv6 limit rate 5/second accept

    # ============================================
    # Rate limiting per-prefix (50 Mbps per peer)
    # ============================================

    # Utwórz mapę: prefix -> meter (per-prefix rate limit)
    # nftables używa meterów dla per-IP/prefix rate limitingu

    if [ "$WG_PREFIX_COUNT" -gt 0 ]; then
        # Chain dla rate limitingu (po firewallu)
        nft add chain inet wg_security rate_limit { type filter hook forward priority 10 \; policy accept \; }

        # Dla każdego prefiksu utwórz osobny meter z limitem 50Mbps
        for PREFIX in $WG_PREFIXES; do
            # Sanitize prefix name for meter (replace : and / with _)
            METER_NAME=$(echo "$PREFIX" | sed 's/[:\\/]/_/g')

            # Egress (upload od klienta) - limit na źródłowym IP
            # 50 Mbps = 6 MB/s (to samo co tc dla SIT/GRE)
            nft add rule inet wg_security rate_limit iifname "wg0" ip6 saddr "$PREFIX" limit rate over 6 mbytes/second drop 2>/dev/null

            # Ingress (download do klienta) - limit na docelowym IP
            nft add rule inet wg_security rate_limit oifname "wg0" ip6 daddr "$PREFIX" limit rate over 6 mbytes/second drop 2>/dev/null
        done

        echo "Skonfigurowano rate limit 50Mbps dla $WG_PREFIX_COUNT prefixów WireGuard"
    fi

    echo "Zakończono konfigurację nftables dla WireGuard"
else
    echo "Interfejs wg0 nie istnieje - pomijam konfigurację WireGuard"
fi

# ============================================
# SEKCJA 3: Zapisanie reguł
# ============================================

mkdir -p /etc/iptables
iptables-save > /etc/iptables/rules.v4
ip6tables-save > /etc/iptables/rules.v6

# Zapisz nftables
nft list ruleset > /etc/nftables.conf 2>/dev/null

echo "Zakończono konfigurację zabezpieczeń"
