package persistence

import (
	"database/sql"
	"fmt"
	"net"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/zinrai/ipam-mvp-go/internal/domain"
	"github.com/zinrai/ipam-mvp-go/internal/infrastructure/db"
)

func TestAllocateIP(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	repo := NewIPAMRepository(db.NewDB(mockDB))

	t.Run("Hostname already in use", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnRows(sqlmock.NewRows([]string{"hostname"}).AddRow("test-host"))
		mock.ExpectRollback()

		_, err := repo.AllocateIP(1, nil, "test-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("Gateway address allocation attempt", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT cidr, gateway FROM networks").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"cidr", "gateway"}).AddRow("192.168.1.0/24", "192.168.1.1"))
		mock.ExpectRollback()

		_, err := repo.AllocateIP(1, net.ParseIP("192.168.1.1"), "test-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("Allocate requested IP", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT cidr, gateway FROM networks").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"cidr", "gateway"}).AddRow("192.168.1.0/24", "192.168.1.1"))
		mock.ExpectQuery("SELECT id, address::text FROM ip_addresses").
			WithArgs(1, "192.168.1.2").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("INSERT INTO ip_addresses").
			WithArgs(1, "192.168.1.2", "test-host").
			WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow(1, "192.168.1.2/32"))
		mock.ExpectCommit()

		ip, err := repo.AllocateIP(1, net.ParseIP("192.168.1.2"), "test-host")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if ip.Address.String() != "192.168.1.2" {
			t.Errorf("expected IP 192.168.1.2, got %s", ip.Address.String())
		}
	})

	t.Run("Allocate first available IP", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT cidr, gateway FROM networks").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"cidr", "gateway"}).AddRow("192.168.1.0/24", "192.168.1.1"))
		mock.ExpectQuery("INSERT INTO ip_addresses").
			WithArgs(1, "192.168.1.2", "test-host").
			WillReturnRows(sqlmock.NewRows([]string{"id", "address"}).AddRow(1, "192.168.1.2/32"))
		mock.ExpectCommit()

		ip, err := repo.AllocateIP(1, nil, "test-host")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if ip.Address.String() != "192.168.1.2" {
			t.Errorf("expected IP 192.168.1.2, got %s", ip.Address.String())
		}
	})

	t.Run("No available IP addresses", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT cidr, gateway FROM networks").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"cidr", "gateway"}).AddRow("192.168.1.0/24", "192.168.1.1"))
		mock.ExpectQuery("INSERT INTO ip_addresses").
			WithArgs(1, sqlmock.AnyArg(), "test-host").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		_, err := repo.AllocateIP(1, nil, "test-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("Database error when checking hostname", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnError(fmt.Errorf("database error"))
		mock.ExpectRollback()

		_, err := repo.AllocateIP(1, nil, "test-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("Invalid gateway IP", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT cidr, gateway FROM networks").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"cidr", "gateway"}).AddRow("192.168.1.0/24", "invalid-ip"))
		mock.ExpectRollback()

		_, err := repo.AllocateIP(1, nil, "test-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("Database error when inserting IP", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "test-host").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT cidr, gateway FROM networks").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"cidr", "gateway"}).AddRow("192.168.1.0/24", "192.168.1.1"))
		mock.ExpectQuery("INSERT INTO ip_addresses").
			WithArgs(1, "192.168.1.2", "test-host").
			WillReturnError(fmt.Errorf("database error"))
		mock.ExpectRollback()

		_, err := repo.AllocateIP(1, net.ParseIP("192.168.1.2"), "test-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func TestCreateNetwork(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	repo := NewIPAMRepository(db.NewDB(mockDB))

	t.Run("Create network successfully", func(t *testing.T) {
		network := &domain.Network{
			CIDR:    "192.168.1.0/24",
			Gateway: net.ParseIP("192.168.1.1"),
		}

		mock.ExpectQuery("INSERT INTO networks").
			WithArgs(network.CIDR, network.Gateway.String()).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		err := repo.CreateNetwork(network)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if network.ID != 1 {
			t.Errorf("expected network ID 1, got %d", network.ID)
		}
	})

	t.Run("Database error when creating network", func(t *testing.T) {
		network := &domain.Network{
			CIDR:    "192.168.1.0/24",
			Gateway: net.ParseIP("192.168.1.1"),
		}

		mock.ExpectQuery("INSERT INTO networks").
			WithArgs(network.CIDR, network.Gateway.String()).
			WillReturnError(fmt.Errorf("database error"))

		err := repo.CreateNetwork(network)
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}

func TestUpdateIPHostname(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer mockDB.Close()

	repo := NewIPAMRepository(db.NewDB(mockDB))

	t.Run("Update hostname successfully", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT network_id FROM ip_addresses").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"network_id"}).AddRow(1))
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "new-host", 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("UPDATE ip_addresses").
			WithArgs("new-host", 1).
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		err := repo.UpdateIPHostname(1, "new-host")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("Hostname already in use", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT network_id FROM ip_addresses").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"network_id"}).AddRow(1))
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "new-host", 1).
			WillReturnRows(sqlmock.NewRows([]string{"hostname"}).AddRow("new-host"))
		mock.ExpectRollback()

		err := repo.UpdateIPHostname(1, "new-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("IP address not found", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT network_id FROM ip_addresses").
			WithArgs(1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err := repo.UpdateIPHostname(1, "new-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})

	t.Run("Database error when updating", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery("SELECT network_id FROM ip_addresses").
			WithArgs(1).
			WillReturnRows(sqlmock.NewRows([]string{"network_id"}).AddRow(1))
		mock.ExpectQuery("SELECT hostname FROM ip_addresses").
			WithArgs(1, "new-host", 1).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectExec("UPDATE ip_addresses").
			WithArgs("new-host", 1).
			WillReturnError(fmt.Errorf("database error"))
		mock.ExpectRollback()

		err := repo.UpdateIPHostname(1, "new-host")
		if err == nil {
			t.Error("expected an error, got nil")
		}
	})
}
