# Chirpy
## Why

__Chirpy__ is an API that allows for users to make requests that allow them to make new chirps, update their account, a webhook to upgrade their account, and checks their authorization for certain tasks.

## How to Get Started (Just using the API)

Make sure to have PostgresSQL installed and running on a machine (it does not have to be the same machine that is running Chirpy).

Clone this repository:

```bash
git clone https://github.com/avgra3/chirpy
```

Create a file at the root of the cloned repository called `.env`. This will contain your environment variables. You will need several parameters:

- `DB_URL`: Something like "postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable"
    - This is where you would change the host address if PostgreSQL is running on a different machine.
- `PLATFORM`: options are "dev" and "release"
- JWT_SECRET: create your secret using your favorite tool
- POLKA_KEY: Our random key to the webhook which checks for a user's __Chirpy Red__ status.

Now, from your terminal run the [buildAndServe.sh](./buildAndServe.sh) from the root directory of the project:

```bash:
./buildAndServe.sh
```

It will run all unit tests before running the server. Now you have Chirpy running!

## API Functions Available
### General HTTP Requests
- `GET /admin/metrics` => Get metrics of the api. Currently, just shows the user __*if*__ they have authorization, how many times the api has responded to requests.
- `POST /admin/reset` => Truncates all data from the `users` and `chirps` tables. Useful for getting started when trying out the api.
- (removed) `POST /api/validate_chirp` => No longer supported. The functionality was to check if a chirp was less than the maximum characters.
- `POST /api/login` => Allow the user to login with a `username` and `password`.
- `POST /api/users` => See all users.
- `POST /api/chirps` => Post a new chirp. Will respond with an error if a user does not have an access token or if the chirp is longer than the 120 character limit. (This inherited the functionality of the `POST /api/validate_chirp` http request.
- `GET /api/chirps` =>  See all chirps.
    - Optional parameters:
        - `sort`: asc or desc the results by the `created_at` field.
        - `author_id`: The UUID of the user who wrote the chirp.
- `GET /api/chirps/{chirpID}` => Get back a specific chirp by using the chirp's UUID.
- `POST /api/refresh` => Refresh the access token for a user.
- `POST /api/revoke` => Revokes a user's access token.
- `PUT /api/users` => Update a user's username or password.
- `DELETE /api/chirps/{chirpID}` => Delete a chirp. You must be the chirp's author and give the corret chirp id.

### Webhooks
- `POST /api/polka/webhooks` => A webhook to allow a user to upgrade their account to "red", a premium feature.
