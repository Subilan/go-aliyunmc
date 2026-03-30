POST {{baseURL}}/user/login
{
    "username": "{{username}}",
    "password": "password"
}
HTTP 200
[Captures]
session-id: cookie "session"

POST {{baseURL}}/task/trigger
[Cookies]
session: {{session-id}}
{
    "type": "test"
}
HTTP 200