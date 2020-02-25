# bad-server
A poorly functioning server to test how your app copes with failures. This server contains an authentication endpoint where you can get a token, then you will need to pass in the token in an 'Authorization' header with a GET request to retrieve data. 

## Quickstart
This app can be run as a standalone go package or via docker to extend potential testing in clustered environments. To run:
``` 
docker build -t bad-server .
docker run -p 8080:8080 bad-server 
```
By default this app will log to stdout. If you run the daemon in detatched mode, you can tail the app logs via `docker logs --follow <image id>`. If you want to log to a persistent volume, the application can target the location by setting the environment variable `LOG_FILE_LOCATION` to the desired output location.


## Status
1. When navigating to the app root, the user will see very basic instructions. These will explain the steps to retrieve the users list, but not in enough detail to do so without explicit instructions provided by this README (intentionally).
2. Calling /auth returns the client_id and a secret key. Users will need to form a valid request by hashing the key correctly. 
3. /users returns the number of users and a list of GUIDs delimited by newlines


## Next
Next steps for the app are:
1. load json data set and deliver data following [RESTful URI patterns][1]  
2. add simple authentication to simulate validation
3. implement mock unstable behaviors (timeouts, error responses, auth failure, incomplete data, etc.)


## Credits
Thanks to:
- https://homework.adhoc.team/noclist/ for the idea
- https://www.callicoder.com/docker-golang-image-container-example/ for the golang server/ docker container inspiration
- https://next.json-generator.com/4yS-UfgXu for providing the sample data

[1]: https://en.wikipedia.org/wiki/Representational_state_transfer#Uniform_interface