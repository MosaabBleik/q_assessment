# 1. Quick Start
## Prequesits
### 1. Ensure Docker and Docker Compose are installed.
### 2. Clone the repository:
```bash
git clone https://github.com/MosaabBleik/q_assessment.git
cd assessment
```
### 3. *Build and run all services*
```bash
docker-compose up
```
### 4. *Access Services:*
- *Products Service:* http://localhost:8080
- *Inventory Service:* http://localhost:8081

# 2. Architecture Overview
graph TD
    A[Products Service: 8080] -->|DB| B(PostgreSQL - Products)
    A -->|Cache| C(Redis)
    D[Inventory Service: 8081] -->|DB| E(PostgreSQL - Inventory)
    D -->|HTTP Request| A

# 3. API Examples
## Products Service:

### Create product
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Lenovo Laptop","description":"gaming laptop","price":3500,"category":"electronics"}'

### List products
curl -X GET http://localhost:8080/api/products

### Product details
curl -X GET http://localhost:8080/api/products/{uuid}

### Delete product
curl -X DELETE http://localhost:8080/api/products/{uuid}

### Update product
curl -X PUT http://localhost:8080/api/products/{uuid} \
  -H "Content-Type: application/json" \
  -d '{"name":"Lenovo Laptop","description":"gaming laptop","price":3500,"category":"electronics"}'

### Bulk update
curl -X PUT http://localhost:8080/api/products/bulk-update \
  -H "Content-Type: application/json" \
  -d '{"products": [{"id":"{uuid}","price":2000}, {"id":"{uuid}","price":2000}]}'

### Search products
curl -X GET http://localhost:8080/api/products/search?q=laptop&category=electronics&min_price=1500&max_price=4000&sort=price


## Inventory Service:

### Add inventory
*NOTE:* You can add more than one inventory with the same product id but for different warehouse locations

curl -X POST http://localhost:8081/api/inventory \
  -H "Content-Type: application/json" \
  -d '{"product_id":"uuid","quantity":100,"warehouse_location":"A1"}'

### Check availability
curl -X GET http://localhost:8081/api/inventory/check-availability \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":"uuid","quantity":5}]}'

# 4. Design Decisions
- *Why separate services?*
The services were separated based on the principle of Single Responsibility Principle (SRP) and Domain Decomposition.

Products Service: Solely responsible for the product catalog (name, price, category).

Inventory Service: Solely responsible for stock levels and location. This separation ensures that changes to the product model (e.g., adding a new field like manufacturer) do not affect the inventory management logic, promoting loose coupling and independent deployment.


- *How services communicate?*
The Inventory Service communicates with Products Service using synchronous HTTP REST calls to check for product existance.


- *Timeout strategy?*
To preventing endless looping if the client cancelled the request, or service fails to response quickly, a mandatory 5-second timeout is applied to all outbound HTTP calls from the Inventory Service to the Products Service.


- *Error handling approach?*
Showing there is an error is not enough for the client, so each error type should be returned with a clear message explaining it.


- *what database technology stack is used?*
*Database:* PostgreSQL was chosen for both services for its reliability and transactional capabilities.

*ORM and Migration:* Instead of using raw SQL and external migration files as suggested, the solution utilizes the GORM library (an ORM for Go) with auto-migration capabilities. This approach simplifies the schema management by automatically creating/updating tables based on the Go data model structs, which is a common practice in rapid development with ORMs. Therefore, the migrations/ folder is present but contains no files, and schema setup is handled at service startup.


# 5. Testing Scenarios
Provide test cases:
## Test 1: Happy path - Product exists, stock available
*Purpose:* Ensure that avialabilty check works when product exists and inventory quantity is sufficient.

*Request*
```json
{
  "items": [
    {"product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b88", "quantity": 28}
  ]
}
```

*Wanted Result*
```json
{
    "available": true,
    "items": [
        {
            "product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b88",
            "requested": 28,
            "available_stock": 50,
            "status": "available",
            "warehouses": [
                {
                    "location": "Riyadh",
                    "quantity": 50
                }
            ]
        }
    ]
}
```

## Test 2: Product not found
*Purpose:* Handle case if product is not found.

*Request*
```json
{
  "items": [
    {"product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b89", "quantity": 28}
  ]
}
```

*Wanted Result*
```json
{
    "available": false,
    "items": [
        {
            "product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b89",
            "requested": 28,
            "available_stock": 0,
            "status": "product_not_found",
            "status_code": 4
        }
    ]
}
```

## Test 3: Insufficient stock

*Request*
```json
{
  "items": [
    {"product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b88", "quantity": 51}
  ]
}
```

*Wanted Result*
```json
{
    "available": false,
    "items": [
        {
            "product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b88",
            "requested": 51,
            "available_stock": 50,
            "status": "insufficient_stock",
            "warehouses": [
                {
                    "location": "Riyadh",
                    "quantity": 50
                }
            ]
        }
    ]
}
```

## Test 4: Products service unavailable (stop it with docker)
*Request*
```json
{
  "items": [
    {"product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b88", "quantity": 51}
  ]
}
```

*Wanted Result*
```json
{
    "available": false,
    "items": [
        {
            "product_id": "e7b3194c-ca2a-4ed5-bc6d-bb8e01175b88",
            "requested": 51,
            "available_stock": 0,
            "status": "products_service_not_available",
            "status_code": 5
        }
    ]
}
```