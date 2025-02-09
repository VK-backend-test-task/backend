package repository

import (
	"backend/domain"
	"context"
	"fmt"
	"net/netip"
	"slices"
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
	PingBefore   *time.Time
	SortProperty domain.ContainerSortProperty
	SortOrder    domain.ContainerOrder
	Limit        int
	Offset       int
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
	if params.Success != nil {
		tx = tx.Where("success = ?", *params.Success)
	}
	if params.ContainerIP != nil {
		tx = tx.Where("container_ip = ?", params.ContainerIP.String())
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
	tx := r.db.Model(&gormPingModel{}).Select("container_ip ip", "max(timestamp) last_ping", "max(case when success then timestamp end) last_success").
		Group("container_ip").Offset(params.Offset).
		Order(clause.OrderByColumn{Column: clause.Column{Name: string(params.SortProperty)}, Desc: params.SortOrder == domain.ContainerSortDesc})
	if params.PingBefore != nil {
		tx = tx.Having("max(timestamp) < ?", params.PingBefore)
	}
	if params.Limit > 0 {
		tx = tx.Limit(params.Limit)
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

		var pLastPing, pLastSuccess *time.Time
		if container.LastPing != "" {
			lastPing, err := time.Parse(time.RFC3339, container.LastPing)
			if err != nil {
				return nil, fmt.Errorf("could not parse last ping time from db: %w", err)
			}
			pLastPing = &lastPing
		}

		if container.LastSuccess != "" {
			lastSuccess, err := time.Parse(time.RFC3339, container.LastSuccess)
			if err != nil {
				return nil, fmt.Errorf("could not parse last successful ping time from db: %w", err)
			}
			pLastSuccess = &lastSuccess
		}

		result[i] = domain.ContainerInfo{
			IP:          containerIP,
			LastPing:    pLastPing,
			LastSuccess: pLastSuccess,
		}
	}
	return result, nil
}

func (r pingRepository) clean() {
	err := r.db.Delete(&gormPingModel{}, "true").Error
	if err != nil {
		panic(err)
	}
}

// this one is intended to be used like a reference implementation of persistent one's
// to compare their output in tests, so it should be very clear to be convincing in its correctness
type inMemoryPingRespository struct {
	pings []domain.Ping
}

// so far so simple
func (r *inMemoryPingRespository) Put(ctx context.Context, pings []domain.Ping) error {
	for _, ping := range pings {
		ping.Timestamp = ping.Timestamp.Truncate(time.Second)
		r.pings = append(r.pings, ping)
	}
	return nil
}

func (r *inMemoryPingRespository) Get(ctx context.Context, params PingGetParams) ([]domain.Ping, error) {
	pings := make([]domain.Ping, 0, len(r.pings))
	// filter
	for _, ping := range r.pings {
		// success mismatch
		if params.Success != nil && ping.Success != *params.Success {
			continue
		}
		// container IP mismatch
		if params.ContainerIP != nil && ping.ContainerIP.Compare(*params.ContainerIP) != 0 {
			continue
		}
		pings = append(pings, ping)
	}
	// sort
	slices.SortFunc(pings, func(a, b domain.Ping) int {
		cmp := a.Timestamp.Compare(b.Timestamp)
		if params.OldestFirst {
			return cmp
		}
		return -cmp
	})
	// cut
	params.Offset = min(params.Offset, len(pings))
	params.Limit = min(params.Limit, len(pings)-params.Offset)
	if params.Limit == 0 {
		params.Limit = len(pings) - params.Offset
	}
	pings = pings[params.Offset : params.Offset+params.Limit]
	return pings, nil
}

func (r *inMemoryPingRespository) Aggregate(ctx context.Context, params PingAggregateParams) ([]domain.ContainerInfo, error) {
	// aggregate
	m := make(map[string]domain.ContainerInfo)
	for _, ping := range r.pings {
		addr := ping.ContainerIP
		k := addr.String()
		curr, ok := m[k]
		if !ok {
			curr = domain.ContainerInfo{IP: addr}
		}
		if curr.LastPing == nil || curr.LastPing.Compare(ping.Timestamp) < 0 {
			curr.LastPing = &ping.Timestamp
		}
		if ping.Success && (curr.LastSuccess == nil || curr.LastSuccess.Compare(ping.Timestamp) < 0) {
			curr.LastSuccess = &ping.Timestamp
		}
		m[k] = curr
	}
	// filter
	containers := make([]domain.ContainerInfo, 0, len(m))
	for _, v := range m {
		if params.PingBefore != nil && v.LastPing.Compare(*params.PingBefore) >= 0 {
			continue
		}
		containers = append(containers, v)
	}
	// sort
	slices.SortFunc(containers, func(a, b domain.ContainerInfo) int {
		var cmp int
		switch params.SortProperty {
		case domain.ContainerSortByLastPing:
			cmp = a.LastPing.Compare(*b.LastPing)
		case domain.ContainerSortByLastSuccess:
			cmp = a.LastSuccess.Compare(*b.LastSuccess)
		default:
			panic("unhandled case")
		}
		if params.SortOrder == domain.ContainerSortAsc {
			return cmp
		}
		return -cmp
	})
	// cut
	params.Offset = min(params.Offset, len(containers))
	params.Limit = min(params.Limit, len(containers)-params.Offset)
	if params.Limit == 0 {
		params.Limit = len(containers) - params.Offset
	}
	containers = containers[params.Offset : params.Offset+params.Limit]
	return containers, nil
}

func (r *inMemoryPingRespository) clean() {
	r.pings = []domain.Ping{}
}

// just to verify that it suits the interface
var _examplePingRepository PingRepository = pingRepository{}
var _exampleMPingRepository PingRepository = &inMemoryPingRespository{}
