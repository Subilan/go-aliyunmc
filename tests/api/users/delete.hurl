POST {{baseURL}}/user/login
{
    "username": "{{username}}",
    "password": "password"
}
HTTP 200
[Captures]
session-id: cookie "session"

DELETE {{baseURL}}/user
[Cookies]
session: {{session-id}}
HTTP 200

POST {{baseURL}}/user/login
{
    "username": "{{username}}",
    "password": "password"
}
HTTP 404