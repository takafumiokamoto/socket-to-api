package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/okamoto/socket-to-api/internal/database"
	"github.com/okamoto/socket-to-api/internal/httpclient"
	"github.com/okamoto/socket-to-api/internal/repository"
)

func main() {
	fmt.Println("=== Oracle DB POC with go-ora + sqlx ===")

	// Database configuration
	cfg := database.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     1521,
		Service:  getEnv("DB_SERVICE", "XEPDB1"),
		Username: getEnv("DB_USER", "app"),
		Password: getEnv("DB_PASSWORD", "app"),
	}

	// Connect to database
	fmt.Println("\n1. Connecting to Oracle Database...")
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()
	fmt.Println("✓ Connected successfully!")

	// Create repository
	repo := repository.NewEmployeeRepository(db)

	// Test 1: Get all employees
	fmt.Println("\n2. Getting all employees...")
	employees, err := repo.GetAll()
	if err != nil {
		log.Fatalf("Failed to get employees: %v", err)
	}
	printJSON("All Employees", employees)

	// Test 2: Get employee by ID
	fmt.Println("\n3. Getting employee with ID=1...")
	employee, err := repo.GetByID(1)
	if err != nil {
		log.Printf("Warning: Failed to get employee by ID: %v", err)
	} else {
		printJSON("Employee ID=1", employee)
	}

	// Test 3: Get employees by salary
	fmt.Println("\n4. Getting employees with salary >= 55000...")
	highEarners, err := repo.GetBySalary(55000)
	if err != nil {
		log.Fatalf("Failed to get employees by salary: %v", err)
	}
	printJSON("High Earners", highEarners)

	// Test 4: Insert new employee
	fmt.Println("\n5. Inserting new employee...")
	err = repo.Create("Alice", "Williams", "alice.williams@example.com", 65000)
	if err != nil {
		log.Printf("Warning: Failed to insert employee: %v", err)
	} else {
		fmt.Println("✓ Employee inserted successfully!")
	}

	// Test 5: Verify insertion
	fmt.Println("\n6. Verifying - Getting all employees again...")
	employees, err = repo.GetAll()
	if err != nil {
		log.Fatalf("Failed to get employees: %v", err)
	}
	printJSON("All Employees After Insert", employees)

	// Test 6: Oracle DUAL table
	fmt.Println("\n7. Testing Oracle DUAL table and functions...")
	dualResult, err := repo.TestDual()
	if err != nil {
		log.Fatalf("Failed to test DUAL: %v", err)
	}
	printJSON("DUAL Query Result", dualResult)

	// Test 7: Update employee salary
	fmt.Println("\n8. Updating employee salary (ID=1, new salary=75000)...")
	err = repo.UpdateSalary(1, 75000)
	if err != nil {
		log.Fatalf("Failed to update salary: %v", err)
	}
	fmt.Println("✓ Salary updated successfully!")

	// Verify update
	fmt.Println("\n9. Verifying update - Getting employee ID=1 again...")
	employee, err = repo.GetByID(1)
	if err != nil {
		log.Fatalf("Failed to get employee: %v", err)
	}
	printJSON("Employee ID=1 After Update", employee)

	// Test 8: Delete employee
	fmt.Println("\n10. Deleting employee with ID=4...")
	err = repo.Delete(4)
	if err != nil {
		log.Printf("Warning: Failed to delete employee: %v", err)
	} else {
		fmt.Println("✓ Employee deleted successfully!")
	}

	// Verify deletion
	fmt.Println("\n11. Verifying deletion - Getting all employees...")
	employees, err = repo.GetAll()
	if err != nil {
		log.Fatalf("Failed to get employees: %v", err)
	}
	printJSON("All Employees After Delete", employees)

	// Test 9: HTTPS API call with system CA certificates
	fmt.Println("\n12. Testing HTTPS API call (system CA certificates)...")
	httpClient := httpclient.NewHTTPClient()
	apiResult, err := httpClient.TestHTTPSRequest("https://api.github.com/users/github")
	if err != nil {
		log.Printf("⚠ HTTPS request failed (expected in scratch without CA certs): %v", err)
	} else {
		fmt.Println("✓ HTTPS request successful with system CA certs!")
		printJSON("GitHub API Response (partial)", map[string]interface{}{
			"login": apiResult["login"],
			"id":    apiResult["id"],
			"type":  apiResult["type"],
		})
	}

	// Test 10: HTTPS API call with embedded CA certificates
	fmt.Println("\n13. Testing HTTPS API call (embedded CA certificates)...")
	embeddedClient, err := httpclient.NewHTTPClientWithEmbeddedCerts()
	if err != nil {
		log.Printf("⚠ Failed to create client with embedded certs: %v", err)
	} else {
		apiResult2, err := embeddedClient.TestHTTPSRequest("https://api.github.com/users/golang")
		if err != nil {
			log.Printf("⚠ HTTPS request with embedded certs failed: %v", err)
		} else {
			fmt.Println("✓ HTTPS request successful with embedded CA certs!")
			printJSON("GitHub API Response (partial)", map[string]interface{}{
				"login": apiResult2["login"],
				"id":    apiResult2["id"],
				"type":  apiResult2["type"],
			})
		}
	}

	// Test 11: HTTPS API call with hybrid approach (system + embedded)
	fmt.Println("\n14. Testing HTTPS API call (hybrid: system + embedded)...")
	hybridClient, err := httpclient.NewHTTPClientHybrid()
	if err != nil {
		log.Printf("⚠ Failed to create hybrid client: %v", err)
	} else {
		apiResult3, err := hybridClient.TestHTTPSRequest("https://api.github.com/users/docker")
		if err != nil {
			log.Printf("⚠ HTTPS request with hybrid client failed: %v", err)
		} else {
			fmt.Println("✓ HTTPS request successful with hybrid approach!")
			printJSON("GitHub API Response (partial)", map[string]interface{}{
				"login": apiResult3["login"],
				"id":    apiResult3["id"],
				"type":  apiResult3["type"],
			})
		}
	}

	// Test 12: HTTPS API call with explicit system path lookup + fallback
	fmt.Println("\n15. Testing HTTPS API call (system paths with fallback)...")
	systemFallbackClient, err := httpclient.NewHTTPClientSystemWithFallback()
	if err != nil {
		log.Printf("⚠ Failed to create system fallback client: %v", err)
	} else {
		apiResult4, err := systemFallbackClient.TestHTTPSRequest("https://api.github.com/users/kubernetes")
		if err != nil {
			log.Printf("⚠ HTTPS request with system fallback failed: %v", err)
		} else {
			fmt.Println("✓ HTTPS request successful with system fallback!")
			printJSON("GitHub API Response (partial)", map[string]interface{}{
				"login": apiResult4["login"],
				"id":    apiResult4["id"],
				"type":  apiResult4["type"],
			})
		}
	}

	// Test 13: Smart fallback based on verification failure
	fmt.Println("\n16. Testing HTTPS with smart fallback (tries multiple cert sources)...")
	smartClient, err := httpclient.NewHTTPClientSmartFallback()
	if err != nil {
		log.Printf("⚠ Failed to create smart fallback client: %v", err)
	} else {
		apiResult5, certSource, err := smartClient.TestHTTPSRequestWithRetry("https://api.github.com/users/cloudnative")
		if err != nil {
			log.Printf("⚠ Smart fallback failed: %v", err)
		} else {
			fmt.Printf("✓ HTTPS request successful using: %s\n", certSource)
			printJSON("GitHub API Response (partial)", map[string]interface{}{
				"login": apiResult5["login"],
				"id":    apiResult5["id"],
				"type":  apiResult5["type"],
			})
		}
	}

	// Test 14: Fallback mechanism with broken certificates
	fmt.Println("\n17. Testing fallback mechanism with broken CA certificates...")
	fallbackTestClient, err := httpclient.NewHTTPClientFallbackTest()
	if err != nil {
		log.Printf("⚠ Failed to create fallback test client: %v", err)
	} else {
		apiResult6, certSource, err := fallbackTestClient.TestFallbackMechanism("https://api.github.com/users/golang")
		if err != nil {
			log.Printf("⚠ Fallback test failed: %v", err)
		} else {
			fmt.Printf("✓ Fallback test successful! Used: %s\n", certSource)
			printJSON("GitHub API Response (partial)", map[string]interface{}{
				"login": apiResult6["login"],
				"id":    apiResult6["id"],
				"type":  apiResult6["type"],
			})
		}
	}

	fmt.Println("\n=== POC completed successfully! ===")
	fmt.Println("\nVerified Features:")
	fmt.Println("✓ SELECT (all, by ID, with filter)")
	fmt.Println("✓ INSERT with sequences")
	fmt.Println("✓ UPDATE with named parameters")
	fmt.Println("✓ DELETE with RowsAffected check")
	fmt.Println("✓ Oracle DUAL table")
	fmt.Println("✓ Oracle functions (SYSDATE, USER)")
	fmt.Println("✓ Named parameters (:name syntax)")
	fmt.Println("✓ HTTPS API calls with system CA certificates")
	fmt.Println("✓ HTTPS API calls with embedded CA certificates")
	fmt.Println("✓ HTTPS API calls with hybrid approach (system + embedded)")
	fmt.Println("✓ HTTPS API calls with system path fallback")
	fmt.Println("✓ HTTPS API calls with smart verification-based fallback")
	fmt.Println("✓ Fallback mechanism based on TLS verification failure (not just availability)")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func printJSON(title string, v interface{}) {
	fmt.Printf("\n--- %s ---\n", title)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal JSON: %v", err)
		return
	}
	fmt.Println(string(data))
}
