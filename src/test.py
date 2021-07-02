#!/usr/bin/env python3

import requests
import json


# x = requests.get("http://localhost:8080/list_printers")
# print(json.dumps(x.json(), indent=2))

body = {
    "userId": "kyle",
    "PrinterId": 1,
    "Duration": 10,
}
x = requests.post("http://localhost:8080/start_print", json=body)
x = requests.post("http://localhost:8080/start_print", json=body)
x = requests.post("http://localhost:8080/start_print", json=body)
x = requests.post("http://localhost:8080/start_print", json=body)
x = requests.post("http://localhost:8080/start_print", json=body)
print(x.status_code, x.text)
body = {
    "userId": "kyle",
    "PrinterId": 2,
    "Duration": 10,
}

x = requests.get("http://localhost:8080/list_prints?userId=kyle")
print(x.status_code, json.dumps(x.json(), indent=2))

# x = requests.get("http://localhost:8080/debug")
