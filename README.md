1. Quick Start
docker-compose up

2. Architecture Overview
Simple diagram:
Products Service (8080) ←──── Inventory Service (8081)
       ↓                              ↓
  PostgreSQL                    PostgreSQL
       ↓
     Redis

3. API Examples
Products Service:

# Create product
curl -X POST http://localhost:8080/api/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Laptop","price":1500,"category":"electronics"}'
Inventory Service:

# Add inventory
curl -X POST http://localhost:8081/api/inventory \
  -H "Content-Type: application/json" \
  -d '{"product_id":"uuid","quantity":100,"warehouse_location":"A1"}'

# Check availability
curl -X GET http://localhost:8081/api/inventory/check-availability \
  -H "Content-Type: application/json" \
  -d '{"items":[{"product_id":"uuid","quantity":5}]}'

4. Design Decisions
•	Why separate services?
•	How services communicate?
•	Timeout strategy?
•	Error handling approach?


5. Testing Scenarios
Provide test cases:
# Test 1: Happy path - Product exists, stock available
# Test 2: Product not found
# Test 3: Insufficient stock
# Test 4: Products service unavailable (stop it with docker)
