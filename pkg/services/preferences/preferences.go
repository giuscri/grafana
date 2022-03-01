package prefs

import (
	"context"

	"github.com/grafana/grafana/pkg/models"
	pstore "github.com/grafana/grafana/pkg/services/preferences/store"
	"github.com/grafana/grafana/pkg/services/sqlstore"
	"github.com/grafana/grafana/pkg/setting"
)

type Manager interface {
	GetPreferencesWithDefaults(context.Context, *models.GetPreferencesWithDefaultsQuery) (*models.Preferences, error)
	GetPreferences(context.Context, *models.GetPreferencesQuery) (*models.Preferences, error)
	SavePreferences(context.Context, *models.SavePreferencesCommand) (*models.Preferences, error)
}

type ManagerImpl struct {
	preferenceStore pstore.Store
}

func ProvideService(cfg *setting.Cfg, sqlStore sqlstore.Store) Manager {
	return &ManagerImpl{
		preferenceStore: pstore.NewPreferencesStore(cfg, sqlStore),
	}
}

func (s *ManagerImpl) GetPreferencesWithDefaults(ctx context.Context, query *models.GetPreferencesWithDefaultsQuery) (*models.Preferences, error) {
	listQuery := &models.ListPreferencesQuery{
		Teams:  query.User.Teams,
		OrgID:  query.User.OrgId,
		UserID: query.User.UserId,
	}
	prefs, err := s.preferenceStore.List(ctx, listQuery)
	if err != nil {
		return nil, err
	}

	res := s.preferenceStore.GetDefaults()
	for _, p := range prefs {
		if p.Theme != "" {
			res.Theme = p.Theme
		}
		if p.Timezone != "" {
			res.Timezone = p.Timezone
		}
		if p.WeekStart != "" {
			res.WeekStart = p.WeekStart
		}
		if p.HomeDashboardId != 0 {
			res.HomeDashboardId = p.HomeDashboardId
		}
	}

	return res, err
}

func (s *ManagerImpl) GetPreferences(ctx context.Context, query *models.GetPreferencesQuery) (*models.Preferences, error) {
	return s.preferenceStore.Get(ctx, query)
}

func (s *ManagerImpl) SavePreferences(ctx context.Context, query *models.SavePreferencesCommand) (*models.Preferences, error) {
	return s.preferenceStore.Set(ctx, query)
}
