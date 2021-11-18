# API Toolkit Golang Client
The API Toolkit golang client is an sdk used to integrate golang web services with APIToolkit. 
It monitors incoming traffic, gathers the requests and sends the request to the apitoolkit servers.

Design decisions:
- Use the gcp SDK to send real time traffic from REST APIs to the gcp topic
