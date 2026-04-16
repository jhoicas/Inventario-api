package crm

import (
	"context"
	"testing"

	"github.com/jhoicas/Inventario-api/internal/domain/entity"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAIAnalyticsRepository struct {
	mock.Mock
}

func (m *mockAIAnalyticsRepository) GetCustomersAtRiskOfChurn(ctx context.Context, daysThreshold int) ([]*entity.CustomerChurnRisk, error) {
	args := m.Called(ctx, daysThreshold)
	if v, ok := args.Get(0).([]*entity.CustomerChurnRisk); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAIAnalyticsRepository) QueryView(ctx context.Context, companyID, sqlQuery string) ([]*entity.AIAnalyticsRow, error) {
	args := m.Called(ctx, companyID, sqlQuery)
	if v, ok := args.Get(0).([]*entity.AIAnalyticsRow); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockAIAnalyticsRepository) RunAggregateQuery(ctx context.Context, companyID, sqlQuery string) (interface{}, error) {
	args := m.Called(ctx, companyID, sqlQuery)
	return args.Get(0), args.Error(1)
}

type mockCRMTaskRepository struct {
	mock.Mock
}

func (m *mockCRMTaskRepository) Create(task *entity.CRMTask) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *mockCRMTaskRepository) GetByID(id string) (*entity.CRMTask, error) {
	args := m.Called(id)
	if v, ok := args.Get(0).(*entity.CRMTask); ok {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCRMTaskRepository) Update(task *entity.CRMTask) error {
	args := m.Called(task)
	return args.Error(0)
}

func (m *mockCRMTaskRepository) ListByCompany(companyID string, status string, limit, offset int) ([]*entity.CRMTask, int64, error) {
	args := m.Called(companyID, status, limit, offset)
	var items []*entity.CRMTask
	if v, ok := args.Get(0).([]*entity.CRMTask); ok {
		items = v
	}
	return items, args.Get(1).(int64), args.Error(2)
}

func (m *mockCRMTaskRepository) CheckTaskExistsForToday(ctx context.Context, companyID, customerName, titlePrefix string) (bool, error) {
	args := m.Called(ctx, companyID, customerName, titlePrefix)
	return args.Bool(0), args.Error(1)
}

type mockRetentionPitchGenerator struct {
	mock.Mock
}

func (m *mockRetentionPitchGenerator) GenerateRetentionPitch(ctx context.Context, customerName, favoriteProduct string, daysInactive int) (string, error) {
	args := m.Called(ctx, customerName, favoriteProduct, daysInactive)
	return args.String(0), args.Error(1)
}

func TestRunChurnPredictionJob_SuccessCreatesTwoTasks(t *testing.T) {
	ctx := context.Background()
	analyticsRepo := new(mockAIAnalyticsRepository)
	taskRepo := new(mockCRMTaskRepository)
	pitchGenerator := new(mockRetentionPitchGenerator)

	worker, err := NewChurnWorker(analyticsRepo, taskRepo, pitchGenerator, nil, 60)
	require.NoError(t, err)

	customers := []*entity.CustomerChurnRisk{
		{
			CompanyID:       "company-1",
			CustomerEmail:   "ana@example.com",
			CustomerName:    "Ana",
			FavoriteProduct: "Producto A",
			DaysInactive:    60,
		},
		{
			CompanyID:       "company-1",
			CustomerName:    "Luis",
			FavoriteProduct: "Producto B",
			DaysInactive:    60,
		},
	}

	analyticsRepo.On("GetCustomersAtRiskOfChurn", mock.Anything, 60).Return(customers, nil).Once()
	taskRepo.On("CheckTaskExistsForToday", mock.Anything, "company-1", "Ana", "Alerta Abandono").Return(false, nil).Once()
	taskRepo.On("CheckTaskExistsForToday", mock.Anything, "company-1", "Luis", "Alerta Abandono").Return(false, nil).Once()
	pitchGenerator.On("GenerateRetentionPitch", mock.Anything, "Ana", "Producto A", 60).Return("Pitch Ana", nil).Once()
	pitchGenerator.On("GenerateRetentionPitch", mock.Anything, "Luis", "Producto B", 60).Return("Pitch Luis", nil).Once()
	taskRepo.On("Create", mock.MatchedBy(func(task *entity.CRMTask) bool {
		return task != nil && task.CompanyID == "company-1" && task.Title == "Alerta Abandono: Ana" && task.Description == "Pitch Ana"
	})).Return(nil).Once()
	taskRepo.On("Create", mock.MatchedBy(func(task *entity.CRMTask) bool {
		return task != nil && task.CompanyID == "company-1" && task.Title == "Alerta Abandono: Luis" && task.Description == "Pitch Luis"
	})).Return(nil).Once()

	worker.RunChurnPredictionJob(ctx)

	analyticsRepo.AssertExpectations(t)
	pitchGenerator.AssertExpectations(t)
	taskRepo.AssertExpectations(t)
	taskRepo.AssertNumberOfCalls(t, "Create", 2)
}

func TestRunChurnPredictionJob_DedupSkipsTaskCreation(t *testing.T) {
	ctx := context.Background()
	analyticsRepo := new(mockAIAnalyticsRepository)
	taskRepo := new(mockCRMTaskRepository)
	pitchGenerator := new(mockRetentionPitchGenerator)

	worker, err := NewChurnWorker(analyticsRepo, taskRepo, pitchGenerator, nil, 60)
	require.NoError(t, err)

	customers := []*entity.CustomerChurnRisk{
		{
			CompanyID:       "company-1",
			CustomerEmail:   "ana@example.com",
			CustomerName:    "Ana",
			FavoriteProduct: "Producto A",
			DaysInactive:    60,
		},
	}

	analyticsRepo.On("GetCustomersAtRiskOfChurn", mock.Anything, 60).Return(customers, nil).Once()
	taskRepo.On("CheckTaskExistsForToday", mock.Anything, "company-1", "Ana", "Alerta Abandono").Return(true, nil).Once()

	worker.RunChurnPredictionJob(ctx)

	analyticsRepo.AssertExpectations(t)
	taskRepo.AssertExpectations(t)
	pitchGenerator.AssertNumberOfCalls(t, "GenerateRetentionPitch", 0)
	taskRepo.AssertNumberOfCalls(t, "Create", 0)
}
