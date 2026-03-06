-- Example: Using DuckDB Integration with Lua
-- This example demonstrates how to use the database functions

-- Load data into the database
local data = {
    users = {
        { id = 1, name = "Alice", email = "alice@example.com", age = 30 },
        { id = 2, name = "Bob", email = "bob@example.com", age = 25 },
        { id = 3, name = "Charlie", email = "charlie@example.com", age = 35 },
    },
    orders = {
        { order_id = 101, user_id = 1, amount = 99.99, created_at = "2024-01-01" },
        { order_id = 102, user_id = 2, amount = 149.99, created_at = "2024-01-02" },
        { order_id = 103, user_id = 1, amount = 49.99, created_at = "2024-01-03" },
    }
}

-- Load the data into DuckDB
local err = stdlib.LoadData(data)
if err then
    print("Error loading data: " .. err)
    return
end

print("Data loaded successfully!")

-- Query 1: Select all users
local users, err = stdlib.ExecuteQuery("SELECT * FROM users")
if err then
    print("Error executing query: " .. err)
    return
end

print("\n--- All Users ---")
for _, user in ipairs(users) do
    print(string.format("ID: %d, Name: %s, Email: %s, Age: %d", 
        user.id, user.name, user.email, user.age))
end

-- Query 2: Join users and orders
local user_orders, err = stdlib.ExecuteQuery(
    "SELECT u.name, u.email, o.order_id, o.amount FROM users u " ..
    "JOIN orders o ON u.id = o.user_id " ..
    "ORDER BY u.name"
)
if err then
    print("Error executing join query: " .. err)
    return
end

print("\n--- User Orders ---")
for _, row in ipairs(user_orders) do
    print(string.format("User: %s (%s), Order ID: %d, Amount: %.2f",
        row.name, row.email, row.order_id, row.amount))
end

-- Query 3: Aggregate query
local summary, err = stdlib.ExecuteQuery(
    "SELECT u.name, COUNT(o.order_id) as order_count, SUM(o.amount) as total_amount " ..
    "FROM users u LEFT JOIN orders o ON u.id = o.user_id " ..
    "GROUP BY u.id, u.name"
)
if err then
    print("Error executing aggregate query: " .. err)
    return
end

print("\n--- User Summary ---")
for _, row in ipairs(summary) do
    if row.order_count and row.total_amount then
        print(string.format("User: %s, Orders: %d, Total: %.2f",
            row.name, row.order_count, row.total_amount))
    else
        print(string.format("User: %s, Orders: 0, Total: 0.00", row.name))
    end
end

print("\nQueries completed successfully!")
