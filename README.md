# URL-Shortener-on-Go
# 1. Architecture: How does it work?

In its simplest form, the service consists of two main parts (endpoints):

- POST `/shorten`: The user sends a long link, the service generates a unique short code (e.g., `abc12`) and saves the ‘code-link’ pair in the database.

- GET `/{code}`: The user clicks on the short link (e.g., `http://localhost:8080/abc12`), the service searches for the code in the database and performs an HTTP redirect to the original long link.

# 2. Choosing a shortening algorithm
Base62 Encoding: We take a unique ID from the database (1, 2, 3...) and convert it to a 62-digit number system (digits 0-9, letters a-z, A-Z). This guarantees uniqueness and no collisions.

# 3. Implementation of MVP (Minimal Viable Product)

The service is written using only the standard Go library. The data will be stored in memory (in a `map`) so as not to be distracted (for now) by databases.

# 4. Final workflow diagram:
  1. Client -> sends a long URL -> Server (stores it in memory).

  2. Server -> returns a short URL -> Client.

  3. Client -> accesses the short URL -> Server (searches in memory).

  4. Server -> tells the browser: ‘Go to this address’ (Redirect) -> Target site.
# 5. Interface (CLI):

-  Data input: Performed via the standard input stream (`stdin`) directly in the terminal.

- Generation: After entering a long link, the programme instantly outputs the finished short link to the console.

- Background process: The web server for redirects runs in parallel and does not block the entry of new links.

- Control: To finish, simply press `Ctrl+C`.
# 6. Monitoring and statistics:

- Live logging: Each action (successful transition or 404 error) is instantly displayed in the console with the time, link code, and target URL.

- Click counter: The system automatically records the number of clicks for each link created.

- `Stat` command: Enter the `stat` command directly into the management console to get a detailed table with a list of all active links and their popularity.

- Basic commands: `stat` - basic statistics, `stat` `[code/Example: Abc123]` - statistics for a specific link, `top <n>` (Example: top 3) - top links by clicks.
#
