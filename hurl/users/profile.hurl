POST {{baseURL}}/user/login
{
    "username": "test",
    "password": "password"
}
HTTP 200
[Captures]
session-id: cookie "session"

GET {{baseURL}}/user/profile?username=test
[Cookies]
session: {{session-id}}
HTTP 200
