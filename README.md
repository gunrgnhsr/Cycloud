# Cycloud

**Resource Exchange Server**

This server is designed to facilitate the exchange of computing resources between Suppliers and Users. Suppliers can offer their spare computing power to Users, who can then use it for their own needs.

## How it works

1. **Registration:** Users and Suppliers register with the server, providing their contact information and resource availability.
2. **Bidding:** Users submit bids for the resources they need, specifying the amount of resources and the price they are willing to pay.
3. **Matching:** The server matches bids with available resources, taking into account the price and the operating costs of the Suppliers.
4. **Exchange:** Once a match is made, the server connects the Supplier and User and allows them to exchange resources directly.

## Features

* **Secure communication:** All communication between the server and clients is encrypted using HTTPS.
* **Authentication and authorization:** Users and Suppliers are authenticated using JWT tokens, which ensures that only authorized users can access resources.
* **Flexible resource types:** The server supports a variety of resource types, including CPU, memory, storage, and network bandwidth.
* **Fair pricing:** The bidding system ensures that Users pay a fair price for the resources they use, taking into account the operating costs of the Suppliers.
* **Easy to use:** The server provides a simple web-based interface for both Users and Suppliers.

## Installation

1. Clone the repository:
    ```sh
    git clone https://github.com/yourusername/Cycloud.git
    cd Cycloud
    ```

2. Install dependencies:
    ```sh
    go mod tidy
    ```

3. Set up environment variables:
    Create a [.env](http://_vscodecontentref_/0) file in the root directory and add the following variables:
    ```env
    DB_HOST=your_db_host
    DB_PORT=your_db_port
    DB_USER=your_db_user
    DB_PASS=your_db_password
    DB_NAME=your_db_name
    DB_SCHEMA=your_db_schema
    ```

4. Run the server:
    ```sh
    go run main.go
    ```

## Usage

1. Register as a User or Supplier.
2. Browse the available resources and submit bids.
3. Once your bid is accepted, connect to the Supplier and start using the resource.

## API Endpoints

- **POST /login:** User login and JWT generation.
- **DELETE /logout:** User logout and JWT invalidation.
- **POST /add-user-resources:** Add a new resource.
- **POST /update-resource-availability/{rid}:** Update resource availability.
- **DELETE /delete-user-resource/{rid}:** Delete a resource.
- **GET /get-user-resources:** Get all resources of a user.
- **GET /get-activly-rented-resources:** Get all actively rented resources of a user.
- **GET /available-resources/{rid}/{direction}:** Get available resources.
- **POST /place-loan-request:** Place a bid for a resource.
- **GET /get-loan-requests:** Get all bids of a user.
- **DELETE /delete-loan-request/{bidId}:** Remove a bid.
- **GET /get-resource/{rid}:** Get resource details.
- **GET /get-info:** Get user information.
- **POST /add-credits:** Add credits to a user's account.
- **POST /accept-connection-offer/{rid}/{token}:** Accept a connection offer.
- **POST /make-connection-offer/{rid}/{token}:** Make a connection offer.
- **GET /get-non-exclusive-tasks:** Get non-exclusive tasks.
- **POST /add-user-task:** Add a new task.
- **DELETE /delete-user-task/{tid}:** Remove a task.
- **POST /update-task-exclusivity/{tid}:** Update task exclusivity.

## License

This project is licensed under the MIT License.

## Technologies used

* Go
* PostgreSQL
