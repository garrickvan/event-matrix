// Copyright 2025 eventmatrix.cn
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		fmt.Printf("failed to connect database: %v", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		age INTEGER NOT NULL
	);
	`

	if _, err := RawSqlExec(db, createTableSQL, nil); err != nil {
		fmt.Printf("Create table failed: %v", err)
	}

	// Insert some data using a transaction
	insertSQL := "INSERT INTO users (name, age) VALUES (@name, @age)"
	i, err := TransactionRawSqlExec(db, insertSQL, map[string]interface{}{"name": "Alice", "age": 25})
	if err != nil {
		fmt.Printf("Transaction failed: %v", err)
	} else {
		fmt.Println("Transaction committed RowsAffected", i)
	}

	// Query data
	selectSQL := "SELECT id, name, age FROM users WHERE age >= @minAge"
	results, err := RawQuerySqlExec(db, selectSQL, map[string]interface{}{"minAge": 25})
	if err != nil {
		fmt.Printf("Query failed: %v", err)
	}

	// Print query results
	fmt.Println("Query results:")
	for _, row := range results {
		fmt.Printf("Row: %v\n", row)
	}

	// Update data using a transaction
	updateSQL := "UPDATE users SET age = @newAge WHERE name = @name"
	i, err = TransactionRawSqlExec(db, updateSQL, map[string]interface{}{"newAge": 35, "name": "Alice"})
	if err != nil {
		fmt.Printf("Update transaction failed: %v", err)
	} else {
		fmt.Println("Update transaction committed RowsAffected", i)
	}

	// Query data after update
	results, err = RawQuerySqlExec(db, selectSQL, map[string]interface{}{"minAge": 20})
	if err != nil {
		fmt.Printf("Query failed: %v", err)
	}

	// Print updated query results
	fmt.Println("Query results after update:")
	for _, row := range results {
		fmt.Printf("Row: %v\n", row)
	}

	// Query total count using rawQuerySqlExec
	countSQL := "SELECT COUNT(*) as total FROM users WHERE age > @minAge"
	countResults, err := RawQuerySqlExec(db, countSQL, map[string]interface{}{"minAge": 20})
	if err != nil {
		fmt.Printf("Count query failed: %v", err)
	} else {
		if len(countResults) > 0 {
			fmt.Println("Count query results:", countResults[0]["total"])
		} else {
			fmt.Println("No results found")
		}
	}

	// Query total count using rawQuerySqlExec
	combinedSQL := `
	SELECT *,
	       (SELECT COUNT(*) FROM users WHERE age > @minAge) AS total
	FROM users
	WHERE age > @minAge
	`

	combinedResults, err := RawQuerySqlExec(db, combinedSQL, map[string]interface{}{"minAge": 20})
	if err != nil {
		fmt.Printf("Count query failed: %v", err)
	} else {
		fmt.Println("Count query results:", combinedResults)
	}

	combinedResults, err = TransactionRawQuerySqlExec(db, combinedSQL, map[string]interface{}{"minAge": 20})
	if err != nil {
		fmt.Printf("Transaction Count query failed: %v", err)
	} else {
		fmt.Println("Transaction Count query results:", combinedResults)
	}

	// Delete db file
	os.Remove("test.db")
}
