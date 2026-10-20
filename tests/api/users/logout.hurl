POST {{baseURL}}/user/login
{
    "username": "{{username}}",
    "password": "password"
}
HTTP 200
[Captures]
session-id: cookie "session"

GET {{baseURL}}/user/logout
[Cookies]
session: {{session-id}}
HTTP 200
