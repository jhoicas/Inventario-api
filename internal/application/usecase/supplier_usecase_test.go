package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/jhoicas/Inventario-api/internal/application/dto"
	"github.com/jhoicas/Inventario-api/internal/domain"
	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/jhoicas/Inventario-api/internal/domain/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSupplierRepository struct {
	createFunc            func(supplier *entity.Supplier) error
	getByIDFunc           func(id string) (*entity.Supplier, error)
	getByCompanyAndNITFun func(companyID, nit string) (*entity.Supplier, error)
	updateFunc            func(supplier *entity.Supplier) error
	listByCompanyFunc     func(companyID, search string, limit, offset int) ([]*entity.Supplier, error)
	setActiveFunc         func(companyID, id string, isActive bool) error
}

func (f *fakeSupplierRepository) Create(supplier *entity.Supplier) error {
	if f.createFunc != nil {
		return f.createFunc(supplier)
	}
	return nil
}

func (f *fakeSupplierRepository) GetByID(id string) (*entity.Supplier, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(id)
	}
	return nil, nil
}

func (f *fakeSupplierRepository) GetByCompanyAndNIT(companyID, nit string) (*entity.Supplier, error) {
	if f.getByCompanyAndNITFun != nil {
		return f.getByCompanyAndNITFun(companyID, nit)
	}
	return nil, nil
}

func (f *fakeSupplierRepository) Update(supplier *entity.Supplier) error {
	if f.updateFunc != nil {
		return f.updateFunc(supplier)
	}
	return nil
}

func (f *fakeSupplierRepository) ListByCompany(companyID, search string, limit, offset int) ([]*entity.Supplier, error) {
	if f.listByCompanyFunc != nil {
		return f.listByCompanyFunc(companyID, search, limit, offset)
	}
	return nil, nil
}

func (f *fakeSupplierRepository) SetActive(companyID, id string, isActive bool) error {
	if f.setActiveFunc != nil {
		return f.setActiveFunc(companyID, id, isActive)
	}
	return nil
}

var _ repository.SupplierRepository = (*fakeSupplierRepository)(nil)

const supplierTestCompanyID = "company-sup-123"

func validSupplierCreateRequest() dto.CreateSupplierRequest {
	return dto.CreateSupplierRequest{
		Name:            "Proveedor 1",
		NIT:             "900123456-7",
		Email:           "proveedor@correo.com",
		Phone:           "3001112233",
		PaymentTermDays: 30,
		LeadTimeDays:    7,
	}
}

func validSupplierEntity(id, companyID string) *entity.Supplier {
	now := time.Now()
	return &entity.Supplier{
		ID:              id,
		CompanyID:       companyID,
		Name:            "Proveedor 1",
		NIT:             "900123456-7",
		Email:           "proveedor@correo.com",
		Phone:           "3001112233",
		PaymentTermDays: 30,
		LeadTimeDays:    7,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func TestSupplierUseCase_Create(t *testing.T) {
	tests := []struct {
		name       string
		companyID  string
		in         dto.CreateSupplierRequest
		repoSetup  func() *fakeSupplierRepository
		wantErr    error
		wantAnyErr bool
	}{
		{
			name:      "Success",
			companyID: supplierTestCompanyID,
			in:        validSupplierCreateRequest(),
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					getByCompanyAndNITFun: func(companyID, nit string) (*entity.Supplier, error) {
						assert.Equal(t, supplierTestCompanyID, companyID)
						assert.Equal(t, "900123456-7", nit)
						return nil, nil
					},
					createFunc: func(supplier *entity.Supplier) error {
						assert.Equal(t, supplierTestCompanyID, supplier.CompanyID)
						assert.Equal(t, "Proveedor 1", supplier.Name)
						assert.NotEmpty(t, supplier.ID)
						return nil
					},
				}
			},
		},
		{
			name:      "InvalidInput_EmptyName",
			companyID: supplierTestCompanyID,
			in: dto.CreateSupplierRequest{
				Name: "",
				NIT:  "900123456-7",
			},
			repoSetup: func() *fakeSupplierRepository { return &fakeSupplierRepository{} },
			wantErr:   domain.ErrInvalidInput,
		},
		{
			name:      "InvalidInput_NegativePaymentTerm",
			companyID: supplierTestCompanyID,
			in: dto.CreateSupplierRequest{
				Name:            "Proveedor",
				NIT:             "900123456-7",
				PaymentTermDays: -1,
			},
			repoSetup: func() *fakeSupplierRepository { return &fakeSupplierRepository{} },
			wantErr:   domain.ErrInvalidInput,
		},
		{
			name:      "Duplicate_NIT",
			companyID: supplierTestCompanyID,
			in:        validSupplierCreateRequest(),
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					getByCompanyAndNITFun: func(_, _ string) (*entity.Supplier, error) {
						return validSupplierEntity("existing", supplierTestCompanyID), nil
					},
				}
			},
			wantErr: domain.ErrDuplicate,
		},
		{
			name:      "RepoCreate_Fails",
			companyID: supplierTestCompanyID,
			in:        validSupplierCreateRequest(),
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					getByCompanyAndNITFun: func(_, _ string) (*entity.Supplier, error) { return nil, nil },
					createFunc:            func(_ *entity.Supplier) error { return errors.New("db error") },
				}
			},
			wantAnyErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewSupplierUseCase(tt.repoSetup())
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
			assert.Equal(t, "Proveedor 1", out.Name)
			assert.Equal(t, "900123456-7", out.NIT)
		})
	}
}

func TestSupplierUseCase_GetByID(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		repoSetup func() *fakeSupplierRepository
		wantNil   bool
		wantErr   bool
	}{
		{
			name: "Success",
			id:   "sup-1",
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					getByIDFunc: func(id string) (*entity.Supplier, error) {
						assert.Equal(t, "sup-1", id)
						return validSupplierEntity("sup-1", supplierTestCompanyID), nil
					},
				}
			},
		},
		{
			name: "NotFound",
			id:   "sup-404",
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{getByIDFunc: func(_ string) (*entity.Supplier, error) { return nil, nil }}
			},
			wantNil: true,
		},
		{
			name: "RepoError",
			id:   "sup-1",
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{getByIDFunc: func(_ string) (*entity.Supplier, error) { return nil, errors.New("db error") }}
			},
			wantErr: true,
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewSupplierUseCase(tt.repoSetup())
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
			assert.Equal(t, "sup-1", out.ID)
		})
	}
}

func TestSupplierUseCase_List(t *testing.T) {
	tests := []struct {
		name      string
		companyID string
		filters   dto.SupplierFilters
		repoSetup func() *fakeSupplierRepository
		wantErr   bool
	}{
		{
			name:      "Success",
			companyID: supplierTestCompanyID,
			filters:   dto.SupplierFilters{Search: "prov", Limit: 20, Offset: 0},
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					listByCompanyFunc: func(companyID, search string, limit, offset int) ([]*entity.Supplier, error) {
						assert.Equal(t, supplierTestCompanyID, companyID)
						assert.Equal(t, "prov", search)
						assert.Equal(t, 20, limit)
						assert.Equal(t, 0, offset)
						return []*entity.Supplier{validSupplierEntity("sup-1", supplierTestCompanyID)}, nil
					},
				}
			},
		},
		{
			name:      "AppliesDefaultLimitAndOffset",
			companyID: supplierTestCompanyID,
			filters:   dto.SupplierFilters{Search: "", Limit: 0, Offset: -1},
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					listByCompanyFunc: func(_ string, _ string, limit, offset int) ([]*entity.Supplier, error) {
						assert.Equal(t, 20, limit)
						assert.Equal(t, 0, offset)
						return []*entity.Supplier{}, nil
					},
				}
			},
		},
		{
			name:      "RepoError",
			companyID: supplierTestCompanyID,
			filters:   dto.SupplierFilters{Limit: 10, Offset: 0},
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{listByCompanyFunc: func(_, _ string, _, _ int) ([]*entity.Supplier, error) {
					return nil, errors.New("db error")
				}}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewSupplierUseCase(tt.repoSetup())
			out, err := uc.List(tt.companyID, tt.filters)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, out)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, out)
		})
	}
}

func TestSupplierUseCase_Update(t *testing.T) {
	newName := "Proveedor Actualizado"
	newNIT := "901000111-2"
	newPayment := 45

	tests := []struct {
		name      string
		id        string
		in        dto.UpdateSupplierRequest
		repoSetup func() *fakeSupplierRepository
		wantErr   error
		wantNil   bool
	}{
		{
			name: "Success",
			id:   "sup-1",
			in: dto.UpdateSupplierRequest{
				Name:            &newName,
				NIT:             &newNIT,
				PaymentTermDays: &newPayment,
			},
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					getByIDFunc: func(_ string) (*entity.Supplier, error) {
						return validSupplierEntity("sup-1", supplierTestCompanyID), nil
					},
					getByCompanyAndNITFun: func(companyID, nit string) (*entity.Supplier, error) {
						assert.Equal(t, supplierTestCompanyID, companyID)
						assert.Equal(t, newNIT, nit)
						return nil, nil
					},
					updateFunc: func(s *entity.Supplier) error {
						assert.Equal(t, newName, s.Name)
						assert.Equal(t, newNIT, s.NIT)
						assert.Equal(t, 45, s.PaymentTermDays)
						return nil
					},
				}
			},
		},
		{
			name: "NotFound",
			id:   "sup-404",
			in:   dto.UpdateSupplierRequest{},
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{getByIDFunc: func(_ string) (*entity.Supplier, error) { return nil, nil }}
			},
			wantNil: true,
		},
		{
			name: "InvalidInput_EmptyName",
			id:   "sup-1",
			in: func() dto.UpdateSupplierRequest {
				empty := ""
				return dto.UpdateSupplierRequest{Name: &empty}
			}(),
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{getByIDFunc: func(_ string) (*entity.Supplier, error) {
					return validSupplierEntity("sup-1", supplierTestCompanyID), nil
				}}
			},
			wantErr: domain.ErrInvalidInput,
			wantNil: true,
		},
		{
			name: "Duplicate_NIT",
			id:   "sup-1",
			in:   dto.UpdateSupplierRequest{NIT: &newNIT},
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					getByIDFunc: func(_ string) (*entity.Supplier, error) {
						return validSupplierEntity("sup-1", supplierTestCompanyID), nil
					},
					getByCompanyAndNITFun: func(_, _ string) (*entity.Supplier, error) {
						return validSupplierEntity("other", supplierTestCompanyID), nil
					},
				}
			},
			wantErr: domain.ErrDuplicate,
			wantNil: true,
		},
		{
			name: "RepoUpdateError",
			id:   "sup-1",
			in:   dto.UpdateSupplierRequest{NIT: &newNIT},
			repoSetup: func() *fakeSupplierRepository {
				return &fakeSupplierRepository{
					getByIDFunc: func(_ string) (*entity.Supplier, error) {
						return validSupplierEntity("sup-1", supplierTestCompanyID), nil
					},
					getByCompanyAndNITFun: func(_, _ string) (*entity.Supplier, error) {
						return nil, nil
					},
					updateFunc: func(_ *entity.Supplier) error {
						return errors.New("db error")
					},
				}
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewSupplierUseCase(tt.repoSetup())
			out, err := uc.Update(tt.id, tt.in)

			if tt.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, out)
				return
			}
			if tt.wantNil {
				if err != nil && tt.name == "RepoUpdateError" {
					assert.Nil(t, out)
					return
				}
				if err == nil {
					assert.Nil(t, out)
					return
				}
			}

			require.NoError(t, err)
			require.NotNil(t, out)
		})
	}
}
