FROM scratch
COPY certs/ca-certificates.crt /etc/ssl/certs/
COPY dns-proxy /dns-proxy
COPY resolv.conf /resolv.conf
ENTRYPOINT ["/dns-proxy"]

