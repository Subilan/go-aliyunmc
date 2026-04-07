POST {{baseURL}}/task/trigger
{
    "type": "create_instance",
    "args": {
        "instanceType": "{{instance}}",
        "zoneId": "{{zone}}",
        "useDefaultVSwitch": true,
        "startWhenCreated": true
    }
}