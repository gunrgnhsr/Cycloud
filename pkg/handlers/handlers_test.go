package handlers

import (
        "bytes"
        "context"
        "encoding/json"
        "net/http"
        "net/http/httptest"
        "testing"

		"github.com/DATA-DOG/go-sqlmock"
        "github.com/gunrgnhsr/Cycloud/pkg/models"
        _ "github.com/lib/pq"
)

func TestCreateResource(t *testing.T) {
        // Create a mock database connection
        db, mock, err := sqlmock.New()
        if err != nil {
                t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
        }
        defer db.Close()

        // Prepare the mock database for the INSERT statement
        mock.ExpectExec("INSERT INTO resources").
                WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
                WillReturnResult(sqlmock.NewResult(1, 1)) // Simulate successful insertion

        // Create a new request with a sample resource
        resource := models.Resource{
                ID:          "test-resource",
                SupplierID:  "test-supplier",
                CPUCores:    4,
                Memory:      8,
                Storage:     500,
                GPU:         "NVIDIA GeForce RTX 3080",
                Bandwidth:   1000,
                CostPerHour: 0.5,
        }
        jsonResource, _ := json.Marshal(resource)
        req, err := http.NewRequest(http.MethodPost, "/resources", bytes.NewBuffer(jsonResource))
        if err != nil {
                t.Fatal(err)
        }

        // Add the mock database to the request context
        ctx := context.WithValue(req.Context(), "db", db)
        req = req.WithContext(ctx)

        // Create a response recorder
        rr := httptest.NewRecorder()
        handler := http.HandlerFunc(CreateResource)

        // Call the handler function
        handler.ServeHTTP(rr, req)

        // Check the status code
        if status := rr.Code; status != http.StatusCreated {
                t.Errorf("handler returned wrong status code: got %v want %v",
                        status, http.StatusCreated)
        }

        // Check if the mock expectations were met
        if err := mock.ExpectationsWereMet(); err != nil {
                t.Errorf("there were unfulfilled expectations: %s", err)
        }
}

func TestGetResource(t *testing.T) {
    // Create a mock database connection
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("an error '%s' was not expected when opening a stub database connection",
 err)
    }
    defer db.Close()

    // Prepare the mock database for the SELECT statement
    expectedResource := models.Resource{
        ID:          "test-resource",
        SupplierID:  "test-supplier",
        CPUCores:    4,
        Memory:      8,
        Storage:     500,
        GPU:         "NVIDIA GeForce RTX 3080",
        Bandwidth:   1000,
        CostPerHour: 0.5,
    }
    rows := sqlmock.NewRows([]string{"id", "supplier_id", "cpu_cores", "memory", "storage", "gpu", "bandwidth", "cost_per_hour"}).
        AddRow(expectedResource.ID, expectedResource.SupplierID, expectedResource.CPUCores, expectedResource.Memory, expectedResource.Storage, expectedResource.GPU, expectedResource.Bandwidth, expectedResource.CostPerHour)
    mock.ExpectQuery("SELECT .* FROM resources WHERE id = \\$1").WithArgs("test-resource").WillReturnRows(rows)

    // Create a new request with the resource ID
    req, err := http.NewRequest(http.MethodGet, "/resources?id=test-resource", nil)
    if err != nil {
        t.Fatal(err)
    }

    // Add the mock database to the request context
    ctx := context.WithValue(req.Context(), "db", db)
    req = req.WithContext(ctx)

    // Create a response recorder
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(GetResource)

    // Call the handler function
    handler.ServeHTTP(rr, req)

    // Check the status code
    if status := rr.Code; status != http.StatusOK {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusOK)

    }

    // Check the response body   

    var actualResource models.Resource
    json.Unmarshal(rr.Body.Bytes(), &actualResource)
    if actualResource != expectedResource {
        t.Errorf("handler returned unexpected body: got %v want %v",
            actualResource, expectedResource)
    }

    // Check if the mock expectations were met
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("there were unfulfilled expectations: %s", err)
    }
}

func TestPlaceBid(t *testing.T) {
    // Create a mock database connection
    db, mock, err := sqlmock.New()
    if err != nil {
        t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
    }
    defer db.Close()

    // Prepare the mock database for the INSERT statement
    mock.ExpectExec("INSERT INTO bids").
        WithArgs(sqlmock.AnyArg(), "test-user", "test-resource", 1.0, 24, "pending").
        WillReturnResult(sqlmock.NewResult(1, 1)) // Simulate successful insertion

    // Create a new request with a sample bid
    bid := models.Bid{
        UserID:     "test-user",
        ResourceID: "test-resource",
        Amount:     1.0,
        Duration:   24,
    }
    jsonBid, _ := json.Marshal(bid)
    req, err := http.NewRequest(http.MethodPost, "/bids", bytes.NewBuffer(jsonBid))
    if err != nil {
        t.Fatal(err)
    }

    // Add the mock database to the request context
    ctx := context.WithValue(req.Context(), "db", db)
    req = req.WithContext(ctx)

    // Create a response recorder
    rr := httptest.NewRecorder()
    handler := http.HandlerFunc(PlaceBid)

    // Call the handler function
    handler.ServeHTTP(rr, req)

    // Check the status code
    if status := rr.Code; status != http.StatusCreated {
        t.Errorf("handler returned wrong status code: got %v want %v",
            status, http.StatusCreated)
    }

    // Check if the mock expectations were met
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("there were unfulfilled expectations: %s", err)
    }
}