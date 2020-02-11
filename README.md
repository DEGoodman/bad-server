# bad-server
A poorly functioning server to test how your app copes with failures

## Status
Currently the application responds to queries at "/" and will respond with "Hello, Guest." thoug you can trigger other names in responses by setting the parameter "?name=<var>" in the request. 

## Next
Next steps for the app are:
1. add simple authentication to simulate API validation
2. implement additional request/response methods, datasets, and validation
3. implement mock unstable behaviors (timeouts, error responses, etc.)

## Notes

This app can be run as a standalone go package or via docker to extend potential testing in clustered environments. To run:
``` 
docker build -t bad-server .
docker run -p 8080:8080 bad-server 
```
By default this app will log to stdout. If you run the daemon in detatched mode, you can tail the app logs via `docker logs --follow <image id>`. If you want to log to a persistent volume, the application can target the location by setting the environment variable `LOG_FILE_LOCATION` to the desired output location.