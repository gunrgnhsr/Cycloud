package pkg

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	"github.com/gunrgnhsr/Cycloud/pkg/models"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
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
type contextKey string

const dbContextKey contextKey = "db"

func GetDBContextKey() contextKey {
	return dbContextKey
}

func getDBSchemaTable(table string) string {
	return os.Getenv("DB_SCHEMA") + "." + table
}

func setTables(db *sql.DB, dbSchema string) error {
	// Create the schema if it doesn't exist
	_, err := db.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", dbSchema))
	if err != nil {
		return err
	}

	tables := []struct {
		name   string
		schema string
	}{
		{
			name: "users",
			schema: `CREATE TABLE ` + dbSchema + `.users (
								uid SERIAL PRIMARY KEY,
								username TEXT NOT NULL UNIQUE,
								password TEXT NOT NULL,
								createdAt TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
						)`,
		},
		{
			name: "wallets",
			schema: `CREATE TABLE ` + dbSchema + `.wallets (
								uid SERIAL PRIMARY KEY,
								credits NUMERIC NOT NULL DEFAULT 0 CHECK (credits >= 0)
						)`,
		},
		{
			name: "tokens",
			schema: `CREATE TABLE ` + dbSchema + `.tokens (
								uid INTEGER NOT NULL,
								token TEXT NOT NULL,
								PRIMARY KEY (uid, token),
								FOREIGN KEY (uid) REFERENCES ` + dbSchema + `.users(uid)
						)`,
		},
		{
			name: "resources",
			schema: `CREATE TABLE ` + dbSchema + `.resources (
								rid SERIAL PRIMARY KEY,
								uid INTEGER NOT NULL,
								cpu_cores INTEGER NOT NULL,
								memory INTEGER NOT NULL,
								storage INTEGER NOT NULL,
								gpu TEXT NOT NULL,
								bandwidth INTEGER NOT NULL,
								cost_per_hour NUMERIC NOT NULL CHECK (cost_per_hour >= 0),
								available BOOLEAN DEFAULT true,
								computing BOOLEAN DEFAULT false,
								createdAt TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
								FOREIGN KEY (uid) REFERENCES ` + dbSchema + `.users(uid)
						)`,
		},
		{
			name: "bids",
			schema: `CREATE TABLE ` + dbSchema + `.bids (
								bid SERIAL PRIMARY KEY,
								uid INTEGER NOT NULL,
								rid INTEGER NOT NULL,
								amount NUMERIC NOT NULL CHECK (amount >= 0),
								duration INTEGER NOT NULL CHECK (duration >= 0),
								status TEXT DEFAULT 'pending',
								computing BOOLEAN DEFAULT false,
								createdAt TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
								FOREIGN KEY (uid) REFERENCES ` + dbSchema + `.users(uid),
								FOREIGN KEY (rid) REFERENCES ` + dbSchema + `.resources(rid)
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
				_, err = db.Exec(fmt.Sprintf("DROP TABLE %s.%s CASCADE", dbSchema, table.name))
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
		Schema:   os.Getenv("DB_SCHEMA"),
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
				// Create a wallet for the new user
				walletTable := getDBSchemaTable("wallets")
				_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (uid) VALUES ($1)", walletTable), uid)
				if err != nil {
					return "", errors.New("failed to create wallet for user")
				}
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
	err := db.QueryRow(fmt.Sprintf("INSERT INTO %s (uid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING rid", table),
		uid, resource.CPUCores, resource.Memory, resource.Storage, resource.GPU, resource.Bandwidth, resource.CostPerHour).Scan(&rid)
	if err != nil {
		return errors.New("failed to insert new resource")
	}
	return nil
}

func UpdateResourceAvailability(db *sql.DB, rid string) (bool, error) {
	var available bool
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("UPDATE %s SET available = NOT available WHERE rid = $1 RETURNING available", table), rid).Scan(&available)
	if err != nil {
		return false, errors.New("failed to update and retrieve resource availability")
	}
	return available, nil
}

func CheckResourceAvailability(db *sql.DB, rid string) (bool, error) {
	var available bool
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("SELECT available FROM %s WHERE rid = $1", table), rid).Scan(&available)
	if err != nil {
		return false, errors.New("failed to check resource availability")
	}
	return available, nil
}

func DeleteResource(db *sql.DB, rid string) error {
	table := getDBSchemaTable("resources")
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE rid = $1", table), rid)
	if err != nil {
		return errors.New("failed to delete resource")
	}
	return nil
}

func GetResourceByID(db *sql.DB, rid string) (models.ResourceWithID, error) {
	var resource models.ResourceWithID
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("SELECT rid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt FROM %s WHERE rid = $1", table), rid).Scan(
		&resource.RID, &resource.Resource.CPUCores, &resource.Resource.Memory, &resource.Resource.Storage, &resource.Resource.GPU, &resource.Resource.Bandwidth, &resource.Resource.CostPerHour, &resource.Resource.Available, &resource.CreatedAt)
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
	rows, err := db.Query(fmt.Sprintf("SELECT rid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt FROM %s WHERE uid = $1 ORDER BY rid", table), uid)
	if err != nil {
		return nil, errors.New("failed to fetch resources")
	}
	defer rows.Close()

	resources := []models.ResourceWithID{}
	for rows.Next() {
		var resource models.ResourceWithID
		err := rows.Scan(&resource.RID, &resource.Resource.CPUCores, &resource.Resource.Memory, &resource.Resource.Storage, &resource.Resource.GPU, &resource.Resource.Bandwidth, &resource.Resource.CostPerHour, &resource.Resource.Available, &resource.CreatedAt)
		if err != nil {
			return nil, errors.New("failed to fetch resources")
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func GetResourceOwner(db *sql.DB, rid string) (string, error) {
	var uid string
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("SELECT uid FROM %s WHERE rid = $1", table), rid).Scan(&uid)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("resource not found")
		} else {
			return "", errors.New("failed to fetch resource")
		}
	}
	return uid, nil
}

func GetNextOrPrevTwentyAvailableResourcesFromGivenRID(db *sql.DB, uid string, rid string, isPrev bool) ([]models.ResourceWithID, error) {
	var operator string
	if isPrev {
		operator = "<"
	} else {
		operator = ">"
	}
	table := getDBSchemaTable("resources")
	rows, err := db.Query(fmt.Sprintf("SELECT rid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt FROM %s WHERE rid %s $1 AND available = true AND uid != $2 LIMIT 20", table, operator), rid, uid)
	if err != nil {
		return nil, errors.New("failed to fetch resources")
	}
	defer rows.Close()

	resources := []models.ResourceWithID{}
	for rows.Next() {
		var resource models.ResourceWithID
		err := rows.Scan(&resource.RID, &resource.Resource.CPUCores, &resource.Resource.Memory, &resource.Resource.Storage, &resource.Resource.GPU, &resource.Resource.Bandwidth, &resource.Resource.CostPerHour, &resource.Resource.Available, &resource.CreatedAt)
		if err != nil {
			return nil, errors.New("failed to fetch resources")
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func InsertNewBid(db *sql.DB, uid string, bid models.Bid) (string, error) {
	var id string

	// Check if the user has enough credits to place the bid
	userCredits, err := GetUserCredits(db, uid)
	if err != nil {
		return "", errors.New("failed to retrieve user credits")
	}

	table := getDBSchemaTable("bids")
	
	// Query the bids table for all the pending and accepted & computing bids of the user
	var totalPendingAmount, totalAcceptedAmount float64
	err = db.QueryRow(fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(CASE WHEN status = 'pending' THEN amount * duration ELSE 0 END), 0) AS total_pending_amount,
			COALESCE(SUM(CASE WHEN status = 'accepted' AND computing = true THEN amount * duration ELSE 0 END), 0) AS total_accepted_amount
		FROM %s 
		WHERE uid = $1`, table), uid).Scan(&totalPendingAmount, &totalAcceptedAmount)
	if err != nil {
		return "", errors.New("failed to retrieve pending and accepted bids")
	}

	var totalAmount float64 = totalPendingAmount + totalAcceptedAmount + bid.Amount * float64(bid.Duration)
	if userCredits < totalAmount {
		return "credit problem", errors.New("insufficient credits to place bid, only " + fmt.Sprintf("%.2f", userCredits) + " credits available and your total bid amount is " + fmt.Sprintf("%.2f", totalAmount))
	}

	err = db.QueryRow(fmt.Sprintf("INSERT INTO %s (uid, rid, amount, duration) VALUES ($1, $2, $3, $4) RETURNING bid", table),
		uid, bid.RID, bid.Amount, bid.Duration).Scan(&id)
	if err != nil {
		return "", errors.New("failed to place bid")
	}
	return id, nil
}

func GetBidOwner(db *sql.DB, bidId string) (string, error) {
	var uid string
	table := getDBSchemaTable("bids")
	err := db.QueryRow(fmt.Sprintf("SELECT uid FROM %s WHERE bid = $1", table), bidId).Scan(&uid)
	if err != nil {
		return "", err
	}
	return uid, nil
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
		err := rows.Scan(&bid.BID, &bid.Bid.RID, &bid.Bid.Amount, &bid.Bid.Duration, &bid.Status, &bid.CreatedAt)
		if err != nil {
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
		err := rows.Scan(&bid.BID, &bid.Bid.RID, &bid.Bid.Amount, &bid.Bid.Duration, &bid.Status, &bid.CreatedAt)
		if err != nil {
			return nil, err
		}
		bids = append(bids, bid)
	}

	return bids, nil
}

func RemoveBidsForResource(db *sql.DB, rid string) error {
	table := getDBSchemaTable("bids")
	_, err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE rid = $1", table), rid)
	if err != nil {
		return err
	}
	return nil
}

func CheckOwnerHaveBidForResource(db *sql.DB, uid string, rid string) (bool, error) {
	var count int
	table := getDBSchemaTable("bids")
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE uid = $1 AND rid = $2", table), uid, rid).Scan(&count)
	if err != nil {
		return false, err
	}
	if count == 0 {
		return false, nil
	}
	return true, nil
}

func GetUserCredits(db *sql.DB, uid string) (float64, error) {
	var amount float64
	table := getDBSchemaTable("wallets")
	err := db.QueryRow(fmt.Sprintf("SELECT credits FROM %s WHERE uid = $1", table), uid).Scan(&amount)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func UpdateUserCredits(db *sql.DB, uid string, amount float64) (float64, error) {
	var updatedAmount float64
	table := getDBSchemaTable("wallets")
	err := db.QueryRow(fmt.Sprintf("UPDATE %s SET credits = credits + $1 WHERE uid = $2 RETURNING credits", table), amount, uid).Scan(&updatedAmount)
	if err != nil {
		return 0, err
	}

	return updatedAmount, nil
}

func GetNumberOfResources(db *sql.DB, uid string) (int, error) {
	var count int
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE uid = $1", table), uid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetNumberOfActiveResources(db *sql.DB, uid string) (int, error) {
	var count int
	table := getDBSchemaTable("resources")
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE uid = $1 AND computing = true", table), uid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetUserOpenBidsTotalAmount(db *sql.DB, uid string) (float64, error) {
	var total float64
	table := getDBSchemaTable("bids")
	err := db.QueryRow(fmt.Sprintf("SELECT COALESCE(SUM(amount), 0) FROM %s WHERE uid = $1 AND status = 'pending'", table), uid).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}

func GetNumberOfAcceptedBidsCurrentlyRunning(db *sql.DB, uid string) (int, error) {
	var count int
	table := getDBSchemaTable("bids")
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE uid = $1 AND status = 'accepted' AND computing = true", table), uid).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}


// bidding package logic
func GetAllAvailableResourcesForBidding(db *sql.DB) ([]models.ResourceWithID, error) {
	table := getDBSchemaTable("resources")
	rows, err := db.Query(fmt.Sprintf("SELECT rid, cpu_cores, memory, storage, gpu, bandwidth, cost_per_hour, available, createdAt FROM %s WHERE available = true AND computing = false ORDER BY rid", table))
	if err != nil {
		return nil, errors.New("failed to fetch resources")
	}
	defer rows.Close()

	resources := []models.ResourceWithID{}
	for rows.Next() {
		var resource models.ResourceWithID
		err := rows.Scan(&resource.RID, &resource.Resource.CPUCores, &resource.Resource.Memory, &resource.Resource.Storage, &resource.Resource.GPU, &resource.Resource.Bandwidth, &resource.Resource.CostPerHour, &resource.Resource.Available, &resource.CreatedAt)
		if err != nil {
			return nil, errors.New("failed to fetch resources")
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func GetMaxBidForResource(db *sql.DB, resourceID string) (models.BidWithUID, error) {
	table := getDBSchemaTable("bids")
	var maxBid models.BidWithUID
	// Set all bids for the resource to 'processing' status
	_, err := db.Exec(fmt.Sprintf("UPDATE %s SET status = 'processing' WHERE rid = $1", table), resourceID)
	if err != nil {
		return models.BidWithUID{}, errors.New("failed to update bids to processing status")
	}
	
	err = db.QueryRow(fmt.Sprintf("SELECT bid, uid, rid, amount, duration, status, createdAt FROM %s WHERE rid = $1 ORDER BY amount DESC, duration DESC LIMIT 1", table), resourceID).Scan(
		&maxBid.BidWithID.BID, &maxBid.UID, &maxBid.BidWithID.Bid.RID, &maxBid.BidWithID.Bid.Amount, &maxBid.BidWithID.Bid.Duration, &maxBid.BidWithID.Status, &maxBid.BidWithID.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.BidWithUID{}, errors.New("no bids found for the resource")
		}
		return models.BidWithUID{}, errors.New("failed to fetch bids")
	}

	// Set the selected bid status to accepted and all other bids for the resource to rejected in one query
	_, err = db.Exec(fmt.Sprintf("UPDATE %s SET status = CASE WHEN bid = $1 THEN 'accepted' ELSE 'rejected' END, computing = CASE WHEN bid = $1 THEN true END WHERE rid = $2", table), maxBid.BidWithID.BID, resourceID)
	if err != nil {
		return models.BidWithUID{}, errors.New("failed to update bids status")
	}

	// Update the resource's computing flag to true
	resourceTable := getDBSchemaTable("resources")
	_, err = db.Exec(fmt.Sprintf("UPDATE %s SET computing = true WHERE rid = $1", resourceTable), resourceID)
	if err != nil {
		return models.BidWithUID{}, errors.New("failed to update resource computing flag")
	}

	return maxBid, nil
}

func FinishCompute(db *sql.DB, resourceID string, bid models.BidWithUID) error {
	// Update the resource's computing flag to false
	resourceTable := getDBSchemaTable("resources")
	_, err := db.Exec(fmt.Sprintf("UPDATE %s SET computing = false WHERE rid = $1", resourceTable), resourceID)
	if err != nil {
		return errors.New("failed to update resource computing flag")
	}
	
	// Update the bid's computing flag to false
	bidTable := getDBSchemaTable("bids")
	_, err = db.Exec(fmt.Sprintf("UPDATE %s SET computing = false WHERE bid = $1", bidTable), bid.BID)
	if err != nil {
		return errors.New("failed to update bid computing flag")
	}

	// Update the resource owner's credits and decrease the credits from the bidding user in one query
	walletTable := getDBSchemaTable("wallets")
	_, err = db.Exec(fmt.Sprintf(`
		UPDATE %s
		SET credits = CASE 
			WHEN uid = $1 THEN credits - $2
			WHEN uid = $3 THEN credits + $2
		END
		WHERE uid IN ($1, $3)`, walletTable), bid.UID, bid.Bid.Amount, bid.Bid.RID)
	if err != nil {
		return errors.New("failed to update credits for resource owner and bidding user")
	}

	return nil
}