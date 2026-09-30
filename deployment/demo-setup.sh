#!/bin/bash

# Demo Setup Script for Samrt Academy
# This creates a sample problem with test cases

API_URL="http://167.71.225.59/api"

echo "========================================="
echo "Samrt Academy - Demo Setup"
echo "========================================="
echo ""

# Step 1: Register admin user
echo "Step 1: Creating admin user..."
REGISTER_RESPONSE=$(curl -s -X POST ${API_URL}/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "admin123",
    "role": "admin"
  }')

echo "Register response: $REGISTER_RESPONSE"
echo ""

# Step 2: Login
echo "Step 2: Logging in..."
LOGIN_RESPONSE=$(curl -s -X POST ${API_URL}/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "Failed to get token. Response: $LOGIN_RESPONSE"
    exit 1
fi

echo "Login successful! Token: ${TOKEN:0:20}..."
echo ""

# Step 3: Create Problem
echo "Step 3: Creating sample problem..."
PROBLEM_RESPONSE=$(curl -s -X POST ${API_URL}/problems \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Sum of Two Numbers",
    "description": "Write a program that reads two integers a and b from input and prints their sum.\n\nInput Format:\nTwo space-separated integers a and b\n\nOutput Format:\nA single integer representing a + b\n\nConstraints:\n-1000 ≤ a, b ≤ 1000",
    "difficulty": "easy",
    "time_limit": 2000,
    "memory_limit": 256000
  }')

PROBLEM_ID=$(echo $PROBLEM_RESPONSE | grep -o '"id":[0-9]*' | cut -d':' -f2)

if [ -z "$PROBLEM_ID" ]; then
    echo "Failed to create problem. Response: $PROBLEM_RESPONSE"
    exit 1
fi

echo "Problem created with ID: $PROBLEM_ID"
echo ""

# Step 4: Add test cases
echo "Step 4: Adding test cases..."

# Test case 1 (sample)
curl -s -X POST ${API_URL}/problems/${PROBLEM_ID}/testcases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "5 3",
    "expected_output": "8",
    "is_sample": true,
    "points": 10
  }' > /dev/null

echo "✓ Test case 1 added (sample)"

# Test case 2 (sample)
curl -s -X POST ${API_URL}/problems/${PROBLEM_ID}/testcases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "10 20",
    "expected_output": "30",
    "is_sample": true,
    "points": 10
  }' > /dev/null

echo "✓ Test case 2 added (sample)"

# Test case 3 (hidden)
curl -s -X POST ${API_URL}/problems/${PROBLEM_ID}/testcases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "-5 15",
    "expected_output": "10",
    "is_sample": false,
    "points": 10
  }' > /dev/null

echo "✓ Test case 3 added (hidden)"

# Test case 4 (hidden)
curl -s -X POST ${API_URL}/problems/${PROBLEM_ID}/testcases \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "input": "0 0",
    "expected_output": "0",
    "is_sample": false,
    "points": 10
  }' > /dev/null

echo "✓ Test case 4 added (hidden)"

echo ""
echo "========================================="
echo "Demo Setup Complete!"
echo "========================================="
echo ""
echo "Admin credentials:"
echo "  Username: admin"
echo "  Password: admin123"
echo ""
echo "Problem created:"
echo "  ID: $PROBLEM_ID"
echo "  Title: Sum of Two Numbers"
echo "  Test cases: 4 (2 sample, 2 hidden)"
echo ""
echo "Next steps:"
echo "1. Open http://167.71.225.59 in your browser"
echo "2. Click on 'Sum of Two Numbers' problem"
echo "3. Write your code in Monaco editor"
echo "4. Click 'Submit' to test against all test cases"
echo ""
echo "Sample solution (Python):"
echo "  a, b = map(int, input().split())"
echo "  print(a + b)"
echo ""
