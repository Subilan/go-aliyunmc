POST {{baseURL}}/user/login
{
    "username": "{{username}}",
    "password": "password"
}
HTTP 200
[Captures]
session-id: cookie "session"

GET {{baseURL}}/task/{{taskId}}/output
[Cookies]
session: {{session-id}}
[Options]
very-verbose: true
HTTP 200