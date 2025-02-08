package repository

import (
	"backend/domain"
	"context"
	"fmt"
	"net/netip"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PingGetParams struct {
	ContainerIP *netip.Addr
	OldestFirst bool
	Success     *bool
	Limit       int
	Offset      int
}

type PingAggregateParams struct {
	PingBefore    *time.Time
	SuccessBefore *time.Time
	SortProperty  *domain.ContainerSortProperty
	SortOrder     *domain.ContainerOrder
	Limit         int
	Offset        int
}

type PingRepository interface {
	Get(ctx context.Context, params PingGetParams) ([]domain.Ping, error)
	Put(ctx context.Context, pings []domain.Ping) error
	Aggregate(ctx context.Context, params PingAggregateParams) ([]domain.ContainerInfo, error)
}

type pingRepository struct {
	db *gorm.DB
}

type gormPingModel struct {
	ID          int
	ContainerIP string
	Timestamp   string
	Success     bool
}

func (gormPingModel) TableName() string {
	return "pings"
}

type gormContainerModel struct {
	IP          string
	LastPing    string
	LastSuccess string
}

func NewPingRepository(db *gorm.DB) PingRepository {
	db.AutoMigrate(&gormPingModel{})
	return pingRepository{db}
}

func (r pingRepository) Get(ctx context.Context, params PingGetParams) ([]domain.Ping, error) {
	gormPings := make([]gormPingModel, 0)
	tx := r.db.
		Offset(params.Offset).
		Order(clause.OrderByColumn{Column: clause.Column{Name: "timestamp"}, Desc: !params.OldestFirst})
	if params.Limit > 0 {
		tx = tx.Limit(params.Limit)
	}
	if params.ContainerIP != nil {
		tx = tx.Where("container = ?", params.ContainerIP.String())
	}
	tx = tx.Find(&gormPings)
	if tx.Error != nil {
		return nil, fmt.Errorf("could not execute transaction: %w", tx.Error)
	}
	result := make([]domain.Ping, len(gormPings))
	for i, gormPing := range gormPings {
		pingContainerIP, err := netip.ParseAddr(gormPing.ContainerIP)
		if err != nil {
			return nil, fmt.Errorf("could not parse IP from db: %w", err)
		}

		pingTimestamp, err := time.Parse(time.RFC3339, gormPing.Timestamp)
		if err != nil {
			return nil, fmt.Errorf("could not parse timestamp from db: %w", err)
		}

		result[i] = domain.Ping{ID: gormPing.ID, ContainerIP: pingContainerIP, Timestamp: pingTimestamp, Success: gormPing.Success}
	}
	return result, nil
}

func (r pingRepository) Put(ctx context.Context, pings []domain.Ping) error {
	gormPings := make([]gormPingModel, len(pings))
	for i, ping := range pings {
		gormPings[i] = gormPingModel{
			ID:          ping.ID,
			ContainerIP: ping.ContainerIP.String(),
			Timestamp:   ping.Timestamp.Format(time.RFC3339),
			Success:     ping.Success,
		}
	}
	return r.db.Create(gormPings).Error
}

func (r pingRepository) Aggregate(ctx context.Context, params PingAggregateParams) ([]domain.ContainerInfo, error) {
	tx := r.db.Model(&gormPingModel{}).Select("max(timestamp) last_ping)", "max(case when success timestamp end) last_success").
		Group("container_ip").Offset(params.Offset)
	if params.Limit > 0 {
		tx = tx.Limit(params.Limit)
	}
	if params.SortOrder != nil {
		tx = tx.Order(clause.OrderByColumn{Column: clause.Column{Name: string(*params.SortProperty)}, Desc: *params.SortOrder == domain.ContainerSortDesc})
	}
	if params.SuccessBefore != nil {
		tx = tx.Having("last_success < ?", params.SuccessBefore)
	}
	if params.PingBefore != nil {
		tx = tx.Having("last_ping < ?", params.PingBefore)
	}
	containers := make([]gormContainerModel, 0)
	tx = tx.Find(&containers)
	if tx.Error != nil {
		return nil, fmt.Errorf("could not execute transaction: %w", tx.Error)
	}
	result := make([]domain.ContainerInfo, len(containers))
	for i, container := range containers {
		containerIP, err := netip.ParseAddr(container.IP)
		if err != nil {
			return nil, fmt.Errorf("could not parse ip from db: %w", err)
		}

		lastPing, err := time.Parse(time.RFC3339, container.LastPing)
		if err != nil {
			return nil, fmt.Errorf("could not parse last ping time from db: %w", err)
		}

		lastSuccess, err := time.Parse(time.RFC3339, container.LastPing)
		if err != nil {
			return nil, fmt.Errorf("could not parse last successful ping time from db: %w", err)
		}

		result[i] = domain.ContainerInfo{
			IP:          containerIP,
			LastPing:    &lastPing,
			LastSuccess: &lastSuccess,
		}
	}
	return result, nil
}

// just to verify that it suits the interface
var _examplePingRepository PingRepository = pingRepository{}
