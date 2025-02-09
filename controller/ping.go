package controller

import (
	"backend/domain"
	"backend/repository"
	"backend/service"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/netip"
	"strconv"
)

type PingController struct {
	svc service.PingService
}

func NewPingsController(svc service.PingService, router *gin.RouterGroup) PingController {
	p := PingController{svc: svc}
	router.GET("/", p.GetPings)
	router.PUT("/", p.PutPings)
	router.GET("/aggregate", p.GetAggregatePings)
	return p
}

func (p PingController) GetPings(c *gin.Context) {
	var pContainerIP *netip.Addr
	rawContainerIP := c.Query("container_ip")
	if rawContainerIP != "" {
		containerIP, err := netip.ParseAddr(rawContainerIP)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid container IP address format"))
			return
		}
		pContainerIP = &containerIP
	}

	rawOldestFirst := c.DefaultQuery("oldest_first", "false")
	oldestFirst := rawOldestFirst == "true"
	if rawOldestFirst != "false" && rawOldestFirst != "true" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("oldest_first should be a boolean"))
		return
	}

	var pSuccess *bool
	rawSuccess := c.Query("success")
	if rawSuccess != "false" && rawSuccess != "true" && rawSuccess != "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("success should be a boolean or empty"))
		return
	}
	if rawSuccess != "" {
		success := rawSuccess == "true"
		pSuccess = &success
	}

	rawLimit := c.DefaultQuery("limit", "0")
	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("limit should be a number"))
		return
	}

	rawOffset := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(rawOffset)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("offset should be a number"))
		return
	}

	result, err := p.svc.Get(c.Request.Context(), repository.PingGetParams{
		ContainerIP: pContainerIP,
		OldestFirst: oldestFirst,
		Success:     pSuccess,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not process your request: %w", err))
		return
	}

	c.JSON(http.StatusOK, result)
}

func (p PingController) PutPings(c *gin.Context) {
	var pings []domain.Ping
	dec := json.NewDecoder(c.Request.Body)
	if err := dec.Decode(&pings); err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid input json"))
		return
	}

	if err := p.svc.Put(c.Request.Context(), pings); err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not process your request: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

func (p PingController) GetAggregatePings(c *gin.Context) {
	rawOldestFirst := c.DefaultQuery("oldest_first", "false")
	oldestFirst := rawOldestFirst == "true"
	if rawOldestFirst != "false" && rawOldestFirst != "true" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("oldest_first should be a boolean"))
		return
	}

	rawLimit := c.DefaultQuery("limit", "0")
	limit, err := strconv.Atoi(rawLimit)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("limit should be a number"))
		return
	}

	rawOffset := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(rawOffset)
	if err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("offset should be a number"))
		return
	}

	result, err := p.svc.Aggregate(c.Request.Context(), repository.PingAggregateParams{
		OldestFirst: oldestFirst,
		Limit:       limit,
		Offset:      offset,
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("could not process your request: %w", err))
		return
	}

	c.JSON(http.StatusOK, result)
}
