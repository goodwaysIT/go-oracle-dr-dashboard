package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goodwaysIT/go-oracle-dr-dashboard/models"
)

// MockApiResponse is a custom response structure for the mock endpoint,
// which includes translated titles along with the data.
type MockApiResponse struct {
	Code      int                    `json:"code"`
	Data      []models.DatabaseStatus `json:"data"`
	Titles    models.TitlesConfig    `json:"titles"`
	Message   string                 `json:"message"`
	Timestamp int64                  `json:"timestamp"`
}

// mockDataHandler generates mock data for 12 databases for screenshots.
func mockDataHandler(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")

	type translations struct {
		Primary         string
		PhysicalStandby string
		Open            string
		ReadOnly        string
		Mounted         string
	}

	trans := map[string]translations{
		"en": {
			Primary:         "PRIMARY",
			PhysicalStandby: "PHYSICAL STANDBY",
			Open:            "OPEN",
			ReadOnly:        "READ ONLY WITH APPLY",
			Mounted:         "MOUNTED",
		},
		"zh": {
			Primary:         "主数据库",
			PhysicalStandby: "物理备库",
			Open:            "读写打开",
			ReadOnly:        "只读应用",
			Mounted:         "已挂载",
		},
		"ja": {
			Primary:         "プライマリ",
			PhysicalStandby: "フィジカル・スタンバイ",
			Open:            "オープン",
			ReadOnly:        "READ ONLY WITH APPLY",
			Mounted:         "マウント済み",
		},
	}

	titleTranslations := map[string]models.TitlesConfig{
		"en": {
			MainTitle:      "Tier-1 Business Oracle DR Monitoring Dashboard (Mock)",
			ProdDataCenter: "Production Data Center",
			DRDataCenter:   "Disaster Recovery Data Center",
		},
		"zh": {
			MainTitle:      "一级业务Oracle容灾监控大盘 (模拟)",
			ProdDataCenter: "生产数据中心",
			DRDataCenter:   "容灾数据中心",
		},
		"ja": {
			MainTitle:      "ティア1ビジネスOracle DR監視ダッシュボード (モック)",
			ProdDataCenter: "本番データセンター",
			DRDataCenter:   "災害復旧データセンター",
		},
	}

	selectedTrans, ok := trans[lang]
	if !ok {
		selectedTrans = trans["en"]
	}
	selectedTitles, ok := titleTranslations[lang]
	if !ok {
		selectedTitles = titleTranslations["en"]
	}

	dbNames := []string{
		"MES_DB", "ERP_DB", "SCM_DB", "WMS_DB", "PLM_DB", "CRM_DB",
		"QMS_DB", "HRM_DB", "FIN_DB", "BI_DB", "OA_DB", "DCS_DB",
	}

	rand.Seed(time.Now().UnixNano())

	dbStatuses := make([]models.DatabaseStatus, 0, len(dbNames))
	for i, name := range dbNames {
		prodRole := selectedTrans.Primary
		disasterRole := selectedTrans.PhysicalStandby
		prodStatus := selectedTrans.Open
		disasterStatus := selectedTrans.ReadOnly
		prodDelay := 0
		disasterDelay := rand.Intn(20) // 0-19 seconds delay

		// Introduce some random variations for realism
		if i%4 == 0 {
			disasterStatus = selectedTrans.Mounted
			disasterDelay = rand.Intn(120) + 60 // 1-3 minutes delay
		}

		status := models.DatabaseStatus{
			Name:                  name,
			LoadBalancerIP:        fmt.Sprintf("172.16.10.%d", 100+i),
			LoadBalancerAlive:     true,
			LoadBalancerPort1521:  true,
			LoadBalancerDbConnect: true,
			Connections:           rand.Intn(150) + 50, // 50-199 connections
			ProductionIP:          fmt.Sprintf("10.10.1.%d", 10+i),
			ProductionAlive:       true,
			ProductionPort1521:    true,
			ProductionDbConnect:   true,
			ProductionStatus:      prodStatus,
			ProductionRole:        prodRole,
			ProductionDgDelay:     prodDelay,
			DisasterIP:            fmt.Sprintf("10.20.1.%d", 10+i),
			DisasterAlive:         true,
			DisasterPort1521:      true,
			DisasterDbConnect:     i%4 != 0, // Simulate a connection issue occasionally
			DisasterStatus:        disasterStatus,
			DisasterRole:          disasterRole,
			DisasterDgDelay:       disasterDelay,
		}
		dbStatuses = append(dbStatuses, status)
	}

	response := MockApiResponse{
		Code:      200,
		Data:      dbStatuses,
		Titles:    selectedTitles,
		Message:   "Mock data generated successfully",
		Timestamp: time.Now().Unix(),
	}

	c.JSON(http.StatusOK, response)
}
