`Chirpy` is mini local clone of twitter.
It is CLI and uses API for communication. 

Local host is configured to 8080 port. 

```
/app/ ->  index.html page 

GET /api/healthz -> status of Chirpy 

GET /admin/metrics -> shows fileserver hits 

POST /api/polka/webhooks -> webhook for third party that informs when user buys paid membership 


GET /api/chiprs -> gets all chirps created by users 

GET /api/chirps?author_id=here_id_of_user -> gets all chirps of only one user by its ID

GET /api/chirps?sort=asc(desc) -> gets all chirps in asc or desc order 

GET /api/chirps/chirpID - gets one chirp by its ID 

POST /api/chirps - creating a chirp 

DELETE /api/chirps/chirpID - deletes one chirp by its ID


POST /api/users - creates new user 

PUT /api/users - updates existing user 

POST /api/login - to log in a user 

POST /admin/reset - deletes users 

POST /api/refresh - refreshs JWT token 

POST /api/revoke - revokes Refresh token 
```

