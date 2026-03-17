package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
)

// ── Fake ProductRepository ──────────────────────────────────────────────────────

type fakeProductRepository struct {
	createFunc             func(product *entity.Product) error
	getByIDFunc            func(id string) (*entity.Product, error)
	getByCompanyAndSKUFunc func(companyID, sku string) (*entity.Product, error)
	updateFunc             func(product *entity.Product) error
	updateCostFunc         func(productID string, cost decimal.Decimal) error
	listByCompanyFunc      func(companyID string, limit, offset int) ([]*entity.Product, error)
	deleteFunc             func(id string) error
}

func (f *fakeProductRepository) Create(product *entity.Product) error {
	if f.createFunc != nil {
		return f.createFunc(product)
	}
	return nil
}

func (f *fakeProductRepository) GetByID(id string) (*entity.Product, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}

func (f *fakeProductRepository) GetByCompanyAndSKU(companyID, sku string) (*entity.Product, error) {
	if f.getByCompanyAndSKUFunc != nil {
		return f.getByCompanyAndSKUFunc(companyID, sku)
	}
	return nil, nil
}

func (f *fakeProductRepository) Update(product *entity.Product) error {
	if f.updateFunc != nil {
		return f.updateFunc(product)
	}
	return nil
}

func (f *fakeProductRepository) UpdateCost(productID string, cost decimal.Decimal) error {
	if f.updateCostFunc != nil {
		return f.updateCostFunc(productID, cost)
	}
	return nil
}

func (f *fakeProductRepository) ListByCompany(companyID string, limit, offset int) ([]*entity.Product, error) {
	if f.listByCompanyFunc != nil {
		return f.listByCompanyFunc(companyID, limit, offset)
	}
	return nil, nil
}

func (f *fakeProductRepository) Delete(id string) error {
	if f.deleteFunc != nil {
		return f.deleteFunc(id)
	}
	return nil
}

// Verificación en tiempo de compilación de que el fake implementa la interfaz.
var _ repository.ProductRepository = (*fakeProductRepository)(nil)

// ── Helpers ────────────────────────────────────────────────────────────────────

const testCompanyID = "company-123"

func validCreateProductRequest() dto.CreateProductRequest {
	return dto.CreateProductRequest{
		SKU:         "SKU-TEST-001",
		Name:        "Producto de prueba",
		Description: "Descripción",
		Price:       decimal.NewFromInt(10000),
		TaxRate:     decimal.NewFromInt(19),
		UNSPSC_Code: "12345678",
		UnitMeasure: "94",
		Attributes:  nil,
	}
}

func validProductEntity(id, companyID string) *entity.Product {
	now := time.Now()
	return &entity.Product{
		ID:           id,
		CompanyID:    companyID,
		SKU:          "SKU-TEST-001",
		Name:         "Producto de prueba",
		Description:  "Descripción",
		Price:        decimal.NewFromInt(10000),
		Cost:         decimal.Zero,
		TaxRate:      decimal.NewFromInt(19),
		UNSPSC_Code:  "12345678",
		UnitMeasure:  "94",
		Attributes:   nil,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// ── Tests Create ────────────────────────────────────────────────────────────────

func TestProductUseCase_Create(t *testing.T) {
	tests := []struct {
		name        string
		companyID   string
		in          dto.CreateProductRequest
		repoSetup   func() *fakeProductRepository
		wantErr     error
		wantAnyErr  bool // si true, solo se exige que err != nil (ej. fallo de repo)
		validateOut func(*testing.T, *dto.ProductResponse)
	}{
		{
			name:      "Success",
			companyID: testCompanyID,
			in:        validCreateProductRequest(),
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByCompanyAndSKUFunc: func(companyID, sku string) (*entity.Product, error) {
						assert.Equal(t, testCompanyID, companyID)
						assert.Equal(t, "SKU-TEST-001", sku)
						return nil, nil // no existe → OK crear
					},
					createFunc: func(product *entity.Product) error {
						assert.Equal(t, testCompanyID, product.CompanyID)
						assert.Equal(t, "SKU-TEST-001", product.SKU)
						assert.Equal(t, "Producto de prueba", product.Name)
						assert.True(t, product.Cost.Equal(decimal.Zero))
						assert.NotEmpty(t, product.ID)
						return nil
					},
				}
			},
			wantErr: nil,
			validateOut: func(t *testing.T, out *dto.ProductResponse) {
				require.NotNil(t, out)
				assert.Equal(t, "SKU-TEST-001", out.SKU)
				assert.Equal(t, "Producto de prueba", out.Name)
				assert.True(t, out.TaxRate.Equal(decimal.NewFromInt(19)))
				assert.True(t, out.Cost.Equal(decimal.Zero))
			},
		},
		{
			name:      "Duplicate_SKUExists",
			companyID: testCompanyID,
			in:        validCreateProductRequest(),
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByCompanyAndSKUFunc: func(_, _ string) (*entity.Product, error) {
						return validProductEntity("existing-id", testCompanyID), nil
					},
				}
			},
			wantErr:     domain.ErrDuplicate,
			validateOut: nil,
		},
		{
			name:      "Success_CustomTaxRate",
			companyID: testCompanyID,
			in: dto.CreateProductRequest{
				SKU:         "SKU-002",
				Name:        "Otro",
				Price:       decimal.Zero,
				TaxRate:     decimal.NewFromInt(10),
				UnitMeasure: "94",
			},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByCompanyAndSKUFunc: func(_, _ string) (*entity.Product, error) { return nil, nil },
					createFunc:             func(_ *entity.Product) error { return nil },
				}
			},
			wantErr: nil,
			validateOut: func(t *testing.T, out *dto.ProductResponse) {
				require.NotNil(t, out)
				assert.True(t, out.TaxRate.Equal(decimal.NewFromInt(10)))
			},
		},
		{
			name:      "InvalidInput_TaxRateAbove100",
			companyID: testCompanyID,
			in: dto.CreateProductRequest{
				SKU:         "SKU-003",
				Name:        "Otro 2",
				Price:       decimal.Zero,
				TaxRate:     decimal.NewFromInt(101),
				UnitMeasure: "94",
			},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByCompanyAndSKUFunc: func(_, _ string) (*entity.Product, error) { return nil, nil },
				}
			},
			wantErr:     domain.ErrInvalidInput,
			validateOut: nil,
		},
		{
			name:      "RepoCreate_Fails",
			companyID: testCompanyID,
			in:        validCreateProductRequest(),
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByCompanyAndSKUFunc: func(_, _ string) (*entity.Product, error) { return nil, nil },
					createFunc: func(_ *entity.Product) error {
						return errors.New("db constraint violation")
					},
				}
			},
			wantAnyErr:  true,
			validateOut: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repoSetup()
			uc := NewProductUseCase(repo)

			out, err := uc.Create(tt.companyID, tt.in)

			if tt.wantAnyErr {
				require.Error(t, err)
				assert.Nil(t, out)
				return
			}
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, out)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, out)
			if tt.validateOut != nil {
				tt.validateOut(t, out)
			}
		})
	}
}

// ── Tests GetByID ───────────────────────────────────────────────────────────────

func TestProductUseCase_GetByID(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		repoSetup   func() *fakeProductRepository
		wantNil     bool
		wantErr     bool
		validateOut func(*testing.T, *dto.ProductResponse)
	}{
		{
			name: "Success",
			id:   "prod-123",
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(id string) (*entity.Product, error) {
						assert.Equal(t, "prod-123", id)
						return validProductEntity(id, testCompanyID), nil
					},
				}
			},
			wantNil: false,
			wantErr: false,
			validateOut: func(t *testing.T, out *dto.ProductResponse) {
				assert.Equal(t, "prod-123", out.ID)
				assert.Equal(t, "SKU-TEST-001", out.SKU)
			},
		},
		{
			name: "NotFound_RepoReturnsNil",
			id:   "prod-999",
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return nil, nil
					},
				}
			},
			wantNil:     true,
			wantErr:     false,
			validateOut: nil,
		},
		{
			name: "Repo_ReturnsError",
			id:   "prod-123",
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantNil:     true,
			wantErr:     true,
			validateOut: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repoSetup()
			uc := NewProductUseCase(repo)

			out, err := uc.GetByID(tt.id)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, out)
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, out)
				return
			}
			require.NotNil(t, out)
			if tt.validateOut != nil {
				tt.validateOut(t, out)
			}
		})
	}
}

// ── Tests List ──────────────────────────────────────────────────────────────────

func TestProductUseCase_List(t *testing.T) {
	tests := []struct {
		name        string
		companyID   string
		limit       int
		offset      int
		repoSetup   func() *fakeProductRepository
		wantErr     bool
		validateOut func(*testing.T, *dto.ProductListResponse)
	}{
		{
			name:      "Success",
			companyID: testCompanyID,
			limit:     20,
			offset:    0,
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					listByCompanyFunc: func(companyID string, limit, offset int) ([]*entity.Product, error) {
						assert.Equal(t, testCompanyID, companyID)
						assert.Equal(t, 20, limit)
						assert.Equal(t, 0, offset)
						return []*entity.Product{validProductEntity("p1", testCompanyID)}, nil
					},
				}
			},
			wantErr: false,
			validateOut: func(t *testing.T, out *dto.ProductListResponse) {
				require.Len(t, out.Items, 1)
				assert.Equal(t, "p1", out.Items[0].ID)
				assert.Equal(t, 20, out.Page.Limit)
				assert.Equal(t, 0, out.Page.Offset)
			},
		},
		{
			name:      "Success_EmptyList",
			companyID: testCompanyID,
			limit:     10,
			offset:    5,
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					listByCompanyFunc: func(_ string, limit, offset int) ([]*entity.Product, error) {
						assert.Equal(t, 10, limit)
						assert.Equal(t, 5, offset)
						return []*entity.Product{}, nil
					},
				}
			},
			wantErr: false,
			validateOut: func(t *testing.T, out *dto.ProductListResponse) {
				assert.Len(t, out.Items, 0)
				assert.Equal(t, 10, out.Page.Limit)
				assert.Equal(t, 5, out.Page.Offset)
			},
		},
		{
			name:      "Repo_ReturnsError",
			companyID: testCompanyID,
			limit:     20,
			offset:    0,
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					listByCompanyFunc: func(_ string, _, _ int) ([]*entity.Product, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantErr:     true,
			validateOut: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repoSetup()
			uc := NewProductUseCase(repo)

			out, err := uc.List(tt.companyID, tt.limit, tt.offset)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, out)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, out)
			if tt.validateOut != nil {
				tt.validateOut(t, out)
			}
		})
	}
}

// ── Tests Update ────────────────────────────────────────────────────────────────

func TestProductUseCase_Update(t *testing.T) {
	newName := "Nombre actualizado"
	newPrice := decimal.NewFromInt(15000)

	tests := []struct {
		name        string
		id          string
		in          dto.UpdateProductRequest
		repoSetup   func() *fakeProductRepository
		wantNil     bool
		wantErr     bool
		errIs       error
		validateOut func(*testing.T, *dto.ProductResponse)
	}{
		{
			name: "Success",
			id:   "prod-123",
			in:   dto.UpdateProductRequest{Name: &newName, Price: &newPrice},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(id string) (*entity.Product, error) {
						return validProductEntity(id, testCompanyID), nil
					},
					updateFunc: func(product *entity.Product) error {
						assert.Equal(t, "Nombre actualizado", product.Name)
						assert.True(t, product.Price.Equal(newPrice))
						return nil
					},
				}
			},
			wantNil: false,
			wantErr: false,
			validateOut: func(t *testing.T, out *dto.ProductResponse) {
				assert.Equal(t, "Nombre actualizado", out.Name)
				assert.True(t, out.Price.Equal(newPrice))
			},
		},
		{
			name: "NotFound_RepoReturnsNilProduct",
			id:   "prod-999",
			in:   dto.UpdateProductRequest{Name: &newName},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return nil, nil
					},
				}
			},
			wantNil:     true,
			wantErr:     false,
			validateOut: nil,
		},
		{
			name: "Success_CustomTaxRate",
			id:   "prod-123",
			in: dto.UpdateProductRequest{
				TaxRate: func() *decimal.Decimal { d := decimal.NewFromInt(7); return &d }(),
			},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return validProductEntity("prod-123", testCompanyID), nil
					},
					updateFunc: func(p *entity.Product) error {
						assert.True(t, p.TaxRate.Equal(decimal.NewFromInt(7)))
						return nil
					},
				}
			},
			wantNil: false,
			wantErr: false,
			validateOut: func(t *testing.T, out *dto.ProductResponse) {
				require.NotNil(t, out)
				assert.True(t, out.TaxRate.Equal(decimal.NewFromInt(7)))
			},
		},
		{
			name: "InvalidInput_TaxRateNegative",
			id:   "prod-123",
			in: dto.UpdateProductRequest{
				TaxRate: func() *decimal.Decimal { d := decimal.NewFromInt(-1); return &d }(),
			},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return validProductEntity("prod-123", testCompanyID), nil
					},
				}
			},
			wantNil: true,
			wantErr: true,
			errIs:   domain.ErrInvalidInput,
		},
		{
			name: "RepoGetByID_ReturnsError",
			id:   "prod-123",
			in:   dto.UpdateProductRequest{Name: &newName},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return nil, errors.New("db error")
					},
				}
			},
			wantNil:     true,
			wantErr:     true,
			validateOut: nil,
		},
		{
			name: "RepoUpdate_ReturnsError",
			id:   "prod-123",
			in:   dto.UpdateProductRequest{Name: &newName},
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					getByIDFunc: func(_ string) (*entity.Product, error) {
						return validProductEntity("prod-123", testCompanyID), nil
					},
					updateFunc: func(_ *entity.Product) error {
						return errors.New("db update failed")
					},
				}
			},
			wantNil:     true,
			wantErr:     true,
			validateOut: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repoSetup()
			uc := NewProductUseCase(repo)

			out, err := uc.Update(tt.id, tt.in)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, out)
				if tt.errIs != nil {
					assert.ErrorIs(t, err, tt.errIs)
				}
				return
			}
			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, out)
				return
			}
			require.NotNil(t, out)
			if tt.validateOut != nil {
				tt.validateOut(t, out)
			}
		})
	}
}

// ── Tests Delete ───────────────────────────────────────────────────────────────

func TestProductUseCase_Delete(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		repoSetup func() *fakeProductRepository
		wantErr   bool
	}{
		{
			name: "Success",
			id:   "prod-123",
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					deleteFunc: func(id string) error {
						assert.Equal(t, "prod-123", id)
						return nil
					},
				}
			},
			wantErr: false,
		},
		{
			name: "Repo_ReturnsError",
			id:   "prod-123",
			repoSetup: func() *fakeProductRepository {
				return &fakeProductRepository{
					deleteFunc: func(_ string) error {
						return errors.New("db error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.repoSetup()
			uc := NewProductUseCase(repo)

			err := uc.Delete(tt.id)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
