POST {{baseURL}}/user/login
[Options]
verbose: true
{
    "username": "{{username}}",
    "password": "password"
}
HTTP 200
[Captures]
session-id: cookie "session"