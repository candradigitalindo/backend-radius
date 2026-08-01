package service

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/candrasyahputra/radius-server/internal/model"
	"github.com/candrasyahputra/radius-server/internal/repository"
)

// fakeCustomerRepo mengimplementasi repository.CustomerRepository via embedding;
// method yang tidak di-override akan panic bila terpanggil (menandakan test perlu diperluas).
type fakeCustomerRepo struct {
	repository.CustomerRepository

	customers      []*model.Customer
	createErrs     []error // error berurutan untuk tiap panggilan Create; habis = sukses
	createCalls    int
	globalUsername map[string]bool // username yang sudah terpakai lintas tenant
}

func (f *fakeCustomerRepo) CountActive(ctx context.Context, tenantID string) (int, error) {
	return len(f.customers), nil
}

func (f *fakeCustomerRepo) CountByCodePrefix(ctx context.Context, tenantID, prefix string) (int, error) {
	count := 0
	for _, c := range f.customers {
		if c.TenantID == tenantID && len(c.CustomerCode) >= len(prefix) && c.CustomerCode[:len(prefix)] == prefix {
			count++
		}
	}
	return count, nil
}

func (f *fakeCustomerRepo) CountByPPPoEPrefix(ctx context.Context, tenantID, prefix string) (int, error) {
	count := 0
	for _, c := range f.customers {
		if c.TenantID == tenantID && len(c.PPPoEUsername) >= len(prefix) && c.PPPoEUsername[:len(prefix)] == prefix {
			count++
		}
	}
	return count, nil
}

func (f *fakeCustomerRepo) FindByPPPoEUsernameGlobal(ctx context.Context, username string) (*model.Customer, error) {
	if f.globalUsername[username] {
		return &model.Customer{PPPoEUsername: username}, nil
	}
	for _, c := range f.customers {
		if c.PPPoEUsername == username {
			return c, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (f *fakeCustomerRepo) Create(ctx context.Context, customer *model.Customer) error {
	f.createCalls++
	if len(f.createErrs) > 0 {
		err := f.createErrs[0]
		f.createErrs = f.createErrs[1:]
		if err != nil {
			return err
		}
	}
	customer.ID = "cust-" + customer.CustomerCode
	saved := *customer
	f.customers = append(f.customers, &saved)
	return nil
}

func (f *fakeCustomerRepo) FindByID(ctx context.Context, tenantID, customerID string) (*model.Customer, error) {
	for _, c := range f.customers {
		if c.TenantID == tenantID && c.ID == customerID {
			return c, nil
		}
	}
	return nil, nil
}

func pgUniqueErr(constraint string) *pgconn.PgError {
	return &pgconn.PgError{Code: "23505", ConstraintName: constraint}
}

func newCreateTestService(repo *fakeCustomerRepo) *CustomerService {
	return &CustomerService{customerRepo: repo}
}

func TestCreateWithManualPPPoECredentials(t *testing.T) {
	repo := &fakeCustomerRepo{}
	svc := newCreateTestService(repo)

	customer, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID:      "t1",
		Name:          "BUDI",
		Phone:         "081234567890",
		PPPoEUsername: "  budi.rumah  ", // spasi harus di-trim
		PPPoEPassword: " rahasia123 ",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if customer.PPPoEUsername != "budi.rumah" {
		t.Errorf("username = %q, ingin %q", customer.PPPoEUsername, "budi.rumah")
	}
	if customer.PPPoEPassword != "rahasia123" {
		t.Errorf("password = %q, ingin %q", customer.PPPoEPassword, "rahasia123")
	}
}

func TestCreateAutoGeneratesWhenEmpty(t *testing.T) {
	repo := &fakeCustomerRepo{}
	svc := newCreateTestService(repo)

	customer, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID: "t1",
		Name:     "SITI",
		Phone:    "081234567890",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	// Format otomatis: YYMMDD + 4 digit terakhir telepon
	if ok, _ := regexp.MatchString(`^\d{6}7890$`, customer.PPPoEUsername); !ok {
		t.Errorf("username otomatis = %q, ingin pola YYMMDD7890", customer.PPPoEUsername)
	}
	if len(customer.PPPoEPassword) != 8 {
		t.Errorf("panjang password otomatis = %d, ingin 8", len(customer.PPPoEPassword))
	}
}

func TestCreateManualUsernameOnlyPasswordAuto(t *testing.T) {
	repo := &fakeCustomerRepo{}
	svc := newCreateTestService(repo)

	customer, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID:      "t1",
		Name:          "ANDI",
		PPPoEUsername: "andi-net",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if customer.PPPoEUsername != "andi-net" {
		t.Errorf("username = %q, ingin %q", customer.PPPoEUsername, "andi-net")
	}
	if len(customer.PPPoEPassword) != 8 {
		t.Errorf("password harus digenerate otomatis (8 karakter), dapat %q", customer.PPPoEPassword)
	}
}

func TestCreateManualUsernameAlreadyTaken(t *testing.T) {
	repo := &fakeCustomerRepo{globalUsername: map[string]bool{"sudah-ada": true}}
	svc := newCreateTestService(repo)

	_, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID:      "t1",
		Name:          "BUDI",
		PPPoEUsername: "sudah-ada",
	})
	if !errors.Is(err, ErrPPPoEUsernameExists) {
		t.Fatalf("err = %v, ingin ErrPPPoEUsernameExists", err)
	}
	if repo.createCalls != 0 {
		t.Errorf("Create repo terpanggil %d kali, seharusnya 0 (ditolak sebelum insert)", repo.createCalls)
	}
}

func TestCreateManualUsernameTakenInOtherTenant(t *testing.T) {
	// Keunikan username bersifat GLOBAL lintas tenant (index customers_pppoe_username_global_key)
	repo := &fakeCustomerRepo{customers: []*model.Customer{
		{ID: "c1", TenantID: "tenant-lain", PPPoEUsername: "punya-orang"},
	}}
	svc := newCreateTestService(repo)

	_, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID:      "t1",
		Name:          "BUDI",
		PPPoEUsername: "punya-orang",
	})
	if !errors.Is(err, ErrPPPoEUsernameExists) {
		t.Fatalf("err = %v, ingin ErrPPPoEUsernameExists", err)
	}
}

func TestCreateManualUsernameInvalid(t *testing.T) {
	cases := []struct {
		name     string
		username string
	}{
		{"mengandung spasi", "budi rumah"},
		{"terlalu pendek", "ab"},
		{"terlalu panjang", "a012345678901234567890123456789012345678901234567890"},
		{"karakter ilegal", "budi;drop"},
		{"non-ascii", "budi√©"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeCustomerRepo{}
			svc := newCreateTestService(repo)
			_, err := svc.Create(context.Background(), CreateCustomerInput{
				TenantID:      "t1",
				Name:          "BUDI",
				PPPoEUsername: tc.username,
			})
			if !errors.Is(err, ErrPPPoEUsernameInvalid) {
				t.Fatalf("username %q: err = %v, ingin ErrPPPoEUsernameInvalid", tc.username, err)
			}
		})
	}
}

func TestCreateManualPasswordInvalid(t *testing.T) {
	cases := []struct {
		name     string
		password string
	}{
		{"mengandung spasi", "pass word"},
		{"terlalu pendek", "abc"},
		{"terlalu panjang", "a012345678901234567890123456789012345678901234567890"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeCustomerRepo{}
			svc := newCreateTestService(repo)
			_, err := svc.Create(context.Background(), CreateCustomerInput{
				TenantID:      "t1",
				Name:          "BUDI",
				PPPoEPassword: tc.password,
			})
			if !errors.Is(err, ErrPPPoEPasswordInvalid) {
				t.Fatalf("password %q: err = %v, ingin ErrPPPoEPasswordInvalid", tc.password, err)
			}
		})
	}
}

func TestCreateRetryKeepsManualUsername(t *testing.T) {
	// Tabrakan customer_code memicu retry; username manual TIDAK boleh diganti otomatis.
	repo := &fakeCustomerRepo{createErrs: []error{pgUniqueErr("customers_customer_code_key")}}
	svc := newCreateTestService(repo)

	customer, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID:      "t1",
		Name:          "BUDI",
		PPPoEUsername: "budi-manual",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if repo.createCalls != 2 {
		t.Errorf("Create repo terpanggil %d kali, ingin 2 (1 gagal + 1 retry)", repo.createCalls)
	}
	if customer.PPPoEUsername != "budi-manual" {
		t.Errorf("username setelah retry = %q, ingin tetap %q", customer.PPPoEUsername, "budi-manual")
	}
}

func TestCreateManualUsernameRaceOnInsert(t *testing.T) {
	// Race: username lolos cek awal tapi bentrok saat insert (constraint pppoe_username)
	repo := &fakeCustomerRepo{createErrs: []error{pgUniqueErr("customers_pppoe_username_global_key")}}
	svc := newCreateTestService(repo)

	_, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID:      "t1",
		Name:          "BUDI",
		PPPoEUsername: "budi-race",
	})
	if !errors.Is(err, ErrPPPoEUsernameExists) {
		t.Fatalf("err = %v, ingin ErrPPPoEUsernameExists", err)
	}
}

func TestCreateAutoUsernameRegeneratedOnRetry(t *testing.T) {
	// Perilaku lama tetap: tanpa username manual, retry menggenerate ulang username.
	repo := &fakeCustomerRepo{
		customers:  []*model.Customer{},
		createErrs: []error{pgUniqueErr("customers_customer_code_key")},
	}
	svc := newCreateTestService(repo)

	customer, err := svc.Create(context.Background(), CreateCustomerInput{
		TenantID: "t1",
		Name:     "SITI",
		Phone:    "0811222333",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if customer.PPPoEUsername == "" {
		t.Error("username otomatis kosong setelah retry")
	}
	if repo.createCalls != 2 {
		t.Errorf("Create repo terpanggil %d kali, ingin 2", repo.createCalls)
	}
}

func TestValidatePPPoEUsername(t *testing.T) {
	valid := []string{"abc", "budi.rumah", "user@isp", "a_b-c.d", "081234567890"}
	for _, u := range valid {
		if !isValidPPPoEUsername(u) {
			t.Errorf("isValidPPPoEUsername(%q) = false, ingin true", u)
		}
	}
	invalid := []string{"", "ab", "a b", "user!", "user#1", "ユーザー"}
	for _, u := range invalid {
		if isValidPPPoEUsername(u) {
			t.Errorf("isValidPPPoEUsername(%q) = true, ingin false", u)
		}
	}
}

func TestValidatePPPoEPassword(t *testing.T) {
	valid := []string{"abcd", "P@ssw0rd!", "1234"}
	for _, p := range valid {
		if !isValidPPPoEPassword(p) {
			t.Errorf("isValidPPPoEPassword(%q) = false, ingin true", p)
		}
	}
	invalid := []string{"", "abc", "a b c d", "pass\tword"}
	for _, p := range invalid {
		if isValidPPPoEPassword(p) {
			t.Errorf("isValidPPPoEPassword(%q) = true, ingin false", p)
		}
	}
}
