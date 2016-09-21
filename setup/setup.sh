#!/bin/sh

if [ ! -f cert.pem ]; then
	echo "This will generate a self-signed certificate to use with cloudmonitor_exporter\nEnter companyname for certificate: "
	read company
	openssl req -x509 -newkey rsa:2048 -keyout key.pem -out ca.pem -days 1080 -nodes -subj '/CN=*/O=$input./C=US' && cp key.pem cert.pem && cat ca.pem >> cert.pem
fi

