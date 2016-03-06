# Build smtpd with GOOS=linux go build
# Build image with docker build -t smtpd .
# Run with docker run --rm -e SMTP_USERNAME=$SMTP_USERNAME -e SMTP_SERVER=$SMTP_SERVER -e SMTP_PORT=$SMTP_PORT -e SMTP_PASSWORD=$SMTP_PASSWORD -p 25:2525 -v /etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt smtpd
FROM scratch
ADD smtpd smtpd
EXPOSE 8080
ENTRYPOINT ["/smtpd"]
