package repository

import (
	"backend/domain"
	"context"
	"database/sql"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/lib/pq"
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
	db *sql.DB
}

func (r *pingRepository) buildStatementForGet(params PingGetParams) string {
	clauses := make([]string, 0)
	if params.ContainerIP != nil {
		clause := fmt.Sprintf(`("container_ip" = %s)`, pq.QuoteLiteral(params.ContainerIP.String()))
		clauses = append(clauses, clause)
	}
	if params.Success != nil {
		if *params.Success {
			clauses = append(clauses, `("success" = TRUE)`)
		} else {
			clauses = append(clauses, `("success" = FALSE)`)
		}
	}

	condition := "TRUE"
	if len(clauses) > 0 {
		condition = strings.Join(clauses, " AND ")
	}

	order := "DESC"
	if params.OldestFirst {
		order = "ASC"
	}

	statementStr := fmt.Sprintf(`SELECT ("id", "container_ip", "timestamp", "success") FROM "pings"
		WHERE %s ORDER BY "timestamp" %s LIMIT %d OFFSET %d;`,
		condition, order, params.Limit, params.Offset)

	return statementStr
}

func (r pingRepository) Get(ctx context.Context, params PingGetParams) ([]domain.Ping, error) {
	queryResult, err := r.db.QueryContext(ctx, r.buildStatementForGet(params))
	if err != nil {
		return nil, fmt.Errorf("could not execute db query: %w", err)
	}
	defer queryResult.Close()

	result := make([]domain.Ping, 0)
	for queryResult.Next() {
		var (
			pingID             int
			pingRawContainerIP string
			pingRawTimestamp   string
			pingSuccess        bool
		)
		if err := queryResult.Scan(&pingID, &pingRawContainerIP, &pingRawTimestamp, &pingSuccess); err != nil {
			return nil, fmt.Errorf("could not read column values from the db: %w", err)
		}

		pingContainerIP, err := netip.ParseAddr(pingRawContainerIP)
		if err != nil {
			return nil, fmt.Errorf("could not parse IP address: %w", err)
		}

		pingTimestamp, err := time.Parse(time.RFC3339, pingRawTimestamp)
		if err != nil {
			return nil, fmt.Errorf("could not parse ping time: %w", err)
		}

		result = append(result, domain.Ping{ID: pingID, ContainerIP: pingContainerIP, Timestamp: pingTimestamp, Success: pingSuccess})
	}

	return result, nil
}

func (r pingRepository) buildPutStatement(pings []domain.Ping) string {
	values := make([]string, 0, len(pings))
	for _, ping := range pings {
		pingContainerIP := ping.ContainerIP.String()
		pingTimestamp := ping.Timestamp.Format(time.RFC3339)
		pingSuccess := "FALSE"
		if ping.Success {
			pingSuccess = "TRUE"
		}
		value := fmt.Sprintf("(%s, %s, %s)", pq.QuoteLiteral(pingContainerIP), pq.QuoteLiteral(pingTimestamp), pingSuccess)
		values = append(values, value)
	}

	statementStr := fmt.Sprintf(`INSERT INTO "pings" ("container_ip", "timestamp", "success") VALUES %s;`, strings.Join(values, ", "))
	return statementStr
}

func (r pingRepository) Put(ctx context.Context, pings []domain.Ping) error {
	if _, err := r.db.ExecContext(ctx, r.buildPutStatement(pings)); err != nil {
		return fmt.Errorf("could not execute statement to save pings in db: %w", err)
	}
	return nil
}

func (r pingRepository) buildStatementForAggregate(params PingAggregateParams) string {
	clauses := make([]string, 0)
	if params.PingBefore != nil {
		clause := fmt.Sprintf(`("last_ping" < %s)`, pq.QuoteLiteral(params.SuccessBefore.UTC().String()))
		clauses = append(clauses, clause)
	}
	if params.SuccessBefore != nil {
		clause := fmt.Sprintf(`("last_success" < %s)`, pq.QuoteLiteral(params.SuccessBefore.UTC().String()))
		clauses = append(clauses, clause)
	}
	if params.SortProperty == nil {
		s := domain.ContainerSortByLastPing
		params.SortProperty = &s
	}
	if params.SortOrder == nil {
		s := domain.ContainerSortDesc
		params.SortOrder = &s
	}
	condition := "TRUE"
	if len(clauses) > 0 {
		condition = strings.Join(clauses, " AND ")
	}
	statementStr := fmt.Sprintf(`
		SELECT MAX("timestamp") "last_ping", MAX(CASE WHEN "success" "timestamp" END) "last_success" FROM "pings"
		GROUP BY "container_ip" HAVING %s ORDER BY %s %s LIMIT %d OFFSET %d;`, condition,
		pq.QuoteLiteral(string(*params.SortProperty)), pq.QuoteLiteral(string(*params.SortOrder)),
		params.Limit, params.Offset)

	return statementStr
}

func (r pingRepository) Aggregate(ctx context.Context, params PingAggregateParams) ([]domain.ContainerInfo, error) {
	queryResult, err := r.db.QueryContext(ctx, r.buildStatementForAggregate(params))
	if err != nil {
		return nil, fmt.Errorf("could not execute db query: %w", err)
	}
	defer queryResult.Close()

	result := make([]domain.ContainerInfo, 0)
	for queryResult.Next() {
		var (
			containerRawIP          string
			containerRawLastPing    string
			containerRawLastSuccess string
		)
		if err := queryResult.Scan(&containerRawIP, &containerRawLastPing, &containerRawLastSuccess); err != nil {
			return nil, fmt.Errorf("could not read column values from the db: %w", err)
		}

		containerIP, err := netip.ParseAddr(containerRawIP)
		if err != nil {
			return nil, fmt.Errorf("could not parse IP from db: %w", err)
		}

		containerLastPing, err := time.Parse(time.RFC3339, containerRawLastPing)
		if err != nil {
			return nil, fmt.Errorf("could not parse time from db: %w", err)
		}

		containerLastSuccess, err := time.Parse(time.RFC3339, containerRawLastPing)
		if err != nil {
			return nil, fmt.Errorf("could not parse time from db: %w", err)
		}

		result = append(result, domain.ContainerInfo{IP: containerIP, LastPing: &containerLastPing, LastSuccess: &containerLastSuccess})
	}

	return result, nil
}

// just to verify that it suits the interface
var _examplePingRepository PingRepository = pingRepository{}
