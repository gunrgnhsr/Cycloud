package pkg

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
        "os"

	"github.com/gunrgnhsr/Cycloud/pkg/models"
	_ "github.com/lib/pq"
        "github.com/joho/godotenv"
)

// DBConfig holds the database configuration parameters.
type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	Schema   string
}

func getTimestamp() string {
	return time.Now().GoString()
}

func getDBSchemaTable(table string) string {
	return os.Getenv("DB_SCHEMA")+"."+table
}

func setTables(db *sql.DB, dbSchema string) error {
	// Create the schema if it doesn't exist
        _, err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s",dbSchema))
        if err != nil {
                return err
        }
        
        
        tables := []struct {
		name   string
		schema string
	}{
		{
			name: "users",
			schema: `CREATE TABLE `+dbSchema+`.users (
                                uid SERIAL PRIMARY KEY,
                                username TEXT NOT NULL UNIQUE,
                                password TEXT NOT NULL
                        )`,
		},
		{
			name: "tokens",
			schema: `CREATE TABLE `+dbSchema+`.tokens (
                                uid INTEGER NOT NULL,
                                token TEXT NOT NULL,
                                PRIMARY KEY (uid, token),
                                FOREIGN KEY (uid) REFERENCES `+dbSchema+`.users(uid)
                        )`,
		},
		{
			name: "resources",
			schema: `CREATE TABLE `+dbSchema+`.resources (
                                rid SERIAL PRIMARY KEY,
                                uid INTEGER NOT NULL,
                                cpu_cores INTEGER NOT NULL,
                                memory INTEGER NOT NULL,
                                storage INTEGER NOT NULL,
                                gpu INTEGER NOT NULL,
                                bandwidth INTEGER NOT NULL,
                                cost_per_hour NUMERIC NOT NULL,
                                available BOOLEAN NOT NULL,
                                createdAt TIMESTAMP NOT NULL,
                                FOREIGN KEY (uid) REFERENCES `+dbSchema+`.users(uid)
                        )`,
		},
		{
			name: "bids",
			schema: `CREATE TABLE `+dbSchema+`.bids (
                                bid SERIAL PRIMARY KEY,
                                uid INTEGER NOT NULL,
                                rid INTEGER NOT NULL,
                                amount NUMERIC NOT NULL,
                                duration INTEGER NOT NULL,
                                status TEXT NOT NULL,
                                createdAt TIMESTAMP NOT NULL,
                                FOREIGN KEY (uid) REFERENCES `+dbSchema+`.users(uid),
                                FOREIGN KEY (rid) REFERENCES `+dbSchema+`.resources(rid)
                        )`,
		},
	}

	for _, table := range tables {
		var exists bool
		err := db.QueryRow(fmt.Sprintf("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = '%s')", table.name)).Scan(&exists)
		if err != nil {
			return err
		}

		if exists {
			// Check if the table has the correct schema
			var columnCount int
			err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = '%s'", table.name)).Scan(&columnCount)
			if err != nil {
				return err
			}

			// If the column count doesn't match, drop and recreate the table
			if columnCount != len(table.schema) {
				_, err = db.Exec(fmt.Sprintf("DROP TABLE %s.%s CASCADE",dbSchema, table.name))
				if err != nil {
					return err
				}
				_, err = db.Exec(table.schema)
				if err != nil {
					return err
				}
			} else {
				// Empty the table -- TODO: This is not a good idea in production
				_, err = db.Exec(fmt.Sprintf("TRUNCATE TABLE %s.%s CASCADE", dbSchema, table.name))
				if err != nil {
					return err
				}
			}
		} else {
			// Create the table
			_, err = db.Exec(table.schema)
			if err != nil {
				return err
			}
		}
	}
	return errors.New("tables created")
}

// NewDB creates a new database connection.
func NewDB() (*sql.DB, error) {
	// Load environment variables from .env
	err := godotenv.Load()
	if err != nil {
			return nil, err
	}

	// Get database configuration from environment variables
	dbConfig := DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASS"),
		DBName:   os.Getenv("DB_NAME"),
		Schema: os.Getenv("DB_SCHEMA"),
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
                dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	err = setTables(db, dbConfig.Schema)
	if err.Error() != "tables created" {
		return nil, err
	}

	return db, nil
}

func CloseDB(db *sql.DB) {
	db.Close()
}

func GetUserOrRegisterIfNotExist(db *sql.DB, username, password string) (string, error) {
	var storedPassword string
	var uid string
	table := getDBSchemaTable("users")
	err := db.QueryRow(fmt.Sprintf("SELECT uid, password FROM %s WHERE username = $1", table), username).Scan(&uid, &storedPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			// Register the new user
			err = db.QueryRow(fmt.Sprintf("INSERT INTO %s (username, password) VALUES ($1, $2) RETURNING uid", table), username, password).Scan(&uid)
			if err != nil {
				return "", errors.New("failed to register user")
			} else {
				return uid, errors.New("user registered")
			}
		} else {
			return "", errors.New("failed to authenticate user")
		}
	} else {
		// Compare the stored password with the provided password
		if storedPassword != password {
			return "", errors.New("invalid username or password")
		}
	}
	return uid, nil
}

func InsertToken(db *sql.DB, uid, token string) error {
	table := getDBSchemaTable("tokens")
	_, err := db.Exec(fmt.Sprintf("INSERT INTO %s (uid, token) VALUES ($1, $2)", table), uid, token)
	if err != nil {
		return errors.New("failed to store token")
	}
	return nil
}

func GetUserIDFromToken(db *sql.DB, token string) (string, error) {
	var uid string
	table := getDBSchemaTable("tokens")
	err := db.QueryRow(fmt.Sprintf("SELECT uid FROM %s WHERE token = $1", table), token).Scan(&uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("token not found")
		}
		return "", errors.New("failed to check token")
	}

	if uid == "" {
		return "", errors.New("token not found")
	}
	return uid, nil
}

func RemoveExpiredToken(db *sql.DB, uid string) error {
	table := getDBSchemaTable("tokens")
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE uid = $1", table), uid)
	if err != nil {
		return errors.New("failed to remove expired token")
	}
	return nil
}

func InsertNewResourse(db *sql.DB, resource models.Resource, uid string) error {
	var rid string
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("INSERT INTO %s (uid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id", table),
		uid, resource.CPUCores, resource.Memory, resource.Storage, resource.GPU, resource.Bandwidth, resource.CostPerHour, resource.Available, getTimestamp()).Scan(&rid)
	if err != nil {
		return errors.New("failed to insert new resource")
	}
	return nil
}

func GetResourceByID(db *sql.DB, id string) (models.ResourceWithID, error) {
	var resource models.ResourceWithID
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("SELECT rid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt FROM %s WHERE id = $1", table), id).Scan(&resource)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.ResourceWithID{}, errors.New("resource not found")
		} else {
			return models.ResourceWithID{}, errors.New("failed to fetch resource")
		}
	}
	return resource, nil
}

func GetUserResources(db *sql.DB, uid string) ([]models.ResourceWithID, error) {
	table := getDBSchemaTable("resources")
	rows, err := db.Query(fmt.Sprintf("SELECT rid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt FROM %s WHERE uid = $1", table), uid)
	if err != nil {
		return nil, errors.New("failed to fetch resources")
	}
	defer rows.Close()

	resources := []models.ResourceWithID{}
	for rows.Next() {
		var resource models.ResourceWithID
		if err := rows.Scan(&resource); err != nil {
			return nil, errors.New("failed to fetch resources")
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func GetResourcesOwner(db *sql.DB, rid string) (string, error) {
	var uid string
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("SELECT uid FROM %s WHERE id = $1", table), rid).Scan(&uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("resource not found")
		} else {
			return "", errors.New("failed to fetch resource")
		}
	}
	return uid, nil
}

func GetNextOrPrevTwentyAvailableResourcesFromGivenRID(db *sql.DB, rid string, isPrev bool) ([]models.ResourceWithID, error) {
	var operator string
	if isPrev {
		operator = "<"
	} else {
		operator = ">"
	}
	table := getDBSchemaTable("resources")
	rows, err := db.Query(fmt.Sprintf("SELECT rid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt FROM %s WHERE rid %s $1 AND available = true LIMIT 20", table, operator), rid)
	if err != nil {
		return nil, errors.New("failed to fetch resources")
	}
	defer rows.Close()

	resources := []models.ResourceWithID{}
	for rows.Next() {
		var resource models.ResourceWithID
		if err := rows.Scan(&resource); err != nil {
			return nil, errors.New("failed to fetch resources")
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func InsertNewBid(db *sql.DB, uid string, bid models.Bid) (string, error) {
	var id string
	table := getDBSchemaTable("bids")
	err := db.QueryRow(fmt.Sprintf("INSERT INTO %s (uid, rid, amount, duration, status, createdAt) VALUES ($1, $2, $3, $4, $5, $6) RETURNING bid", table),
		uid, bid.RID, bid.Amount, bid.Duration, bid.Status, getTimestamp()).Scan(&id)
	if err != nil {
		return "", errors.New("failed to place bid")
	}
	return id, nil
}

func GetBidByID(db *sql.DB, id string) (models.BidWithID, error) {
	var bid models.BidWithID
	table := getDBSchemaTable("bids")
	err := db.QueryRow(fmt.Sprintf("SELECT bid, uid, rid, amount, duration, status, createdAt FROM %s WHERE bid = $1", table), id).Scan(&bid)
	if err != nil {
		return models.BidWithID{}, err
	}
	return bid, nil
}

func RemoveBid(db *sql.DB, id string) error {
	table := getDBSchemaTable("bids")
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE bid = $1", table), id)
	if err != nil {
		return err
	}
	return nil
}

func GetUserBids(db *sql.DB, uid string) ([]models.BidWithID, error) {
	table := getDBSchemaTable("bids")
	rows, err := db.Query(fmt.Sprintf("SELECT bid, rid, amount, duration, status, createdAt FROM %s WHERE uid = $1 ORDER BY rid", table), uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bids := []models.BidWithID{}
	for rows.Next() {
		var bid models.BidWithID
		if err := rows.Scan(&bid); err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}

	return bids, nil
}

func GetBidsForResource(db *sql.DB, rid string) ([]models.BidWithID, error) {
	table := getDBSchemaTable("bids")
	rows, err := db.Query(fmt.Sprintf("SELECT bid, uid, rid, amount, duration, status, createdAt FROM %s WHERE rid = $1", table), rid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bids := []models.BidWithID{}
	for rows.Next() {
		var bid models.BidWithID
		if err := rows.Scan(&bid); err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}

	return bids, nil
}
