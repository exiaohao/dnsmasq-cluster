FROM andyshinn/dnsmasq:2.78

RUN echo "nameserver 8.8.4.4" > /etc/resolv.dnsmasq

ENTRYPOINT [ "dnsmasq", "-k", "-q", "-p", "53" ]