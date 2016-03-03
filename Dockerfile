FROM scratch
ADD smtpd smtpd
EXPOSE 8080
ENTRYPOINT ["/smtpd"]
