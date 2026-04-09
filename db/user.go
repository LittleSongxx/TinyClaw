package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/param"
)

type User struct {
	ID           int64            `json:"id"`
	UserId       string           `json:"user_id"`
	Token        int              `json:"token"`
	UpdateTime   int64            `json:"update_time"`
	CreateTime   int64            `json:"create_time"`
	AvailToken   int              `json:"avail_token"`
	Unlimited    bool             `json:"unlimited"`
	LLMConfig    string           `json:"llm_config"`
	LLMConfigRaw *param.LLMConfig `json:"llm_config_raw"`
}

type UserQuotaMetric struct {
	ID             int64   `json:"id"`
	UserID         string  `json:"user_id"`
	Token          int     `json:"token"`
	AvailToken     int     `json:"avail_token"`
	RemainingToken int     `json:"remaining_token"`
	UsageRate      float64 `json:"usage_rate"`
	Unlimited      bool    `json:"unlimited"`
	CreateTime     int64   `json:"create_time"`
	UpdateTime     int64   `json:"update_time"`
}

type UserQuotaBucket struct {
	Label string `json:"label"`
	Min   int    `json:"min"`
	Max   int    `json:"max"`
	Count int    `json:"count"`
}

type UserQuotaSummary struct {
	TotalUsers        int     `json:"total_users"`
	TotalUsedToken    int     `json:"total_used_token"`
	TotalRemainToken  int     `json:"total_remaining_token"`
	TotalQuotaToken   int     `json:"total_quota_token"`
	AverageUsageRate  float64 `json:"average_usage_rate"`
	UnlimitedUsers    int     `json:"unlimited_users"`
}

type UserQuotaStats struct {
	Summary         UserQuotaSummary  `json:"summary"`
	TopUsed         []UserQuotaMetric `json:"top_used"`
	LowestRemaining []UserQuotaMetric `json:"lowest_remaining"`
	Distribution    []UserQuotaBucket `json:"distribution"`
	List            []UserQuotaMetric `json:"list"`
	Total           int               `json:"total"`
	Page            int               `json:"page"`
	PageSize        int               `json:"page_size"`
	SortBy          string            `json:"sort_by"`
	UserID          string            `json:"user_id,omitempty"`
}

// InsertUser insert user data
func InsertUser(userId string, llmConfig string) (int64, error) {
	userInfo, err := GetUserByID(userId)
	if err != nil {
		return 0, err
	}
	if userInfo != nil && userInfo.ID != 0 {
		return userInfo.ID, nil
	}

	// insert data
	insertSQL := `INSERT INTO users (user_id, llm_config, update_time, create_time, avail_token, from_bot) VALUES (?, ?, ?, ?, ?, ?)`
	result, err := DB.Exec(insertSQL, userId, llmConfig, time.Now().Unix(), time.Now().Unix(), conf.BaseConfInfo.TokenPerUser, conf.BaseConfInfo.BotName)
	if err != nil {
		return 0, err
	}

	// get last insert id
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetUserByID get user by userId
func GetUserByID(userId string) (*User, error) {
	// select one use base on name
	querySQL := `SELECT id, user_id, llm_config, token, avail_token, update_time, create_time FROM users WHERE user_id = ?`
	row := DB.QueryRow(querySQL, userId)

	// scan row get result
	var user User
	err := row.Scan(&user.ID, &user.UserId, &user.LLMConfig, &user.Token, &user.AvailToken, &user.UpdateTime, &user.CreateTime)
	if err != nil {
		if err == sql.ErrNoRows {
			// 如果没有找到数据，返回 nil
			return nil, nil
		}
		return nil, err
	}

	if user.LLMConfig != "" {
		err := json.Unmarshal([]byte(user.LLMConfig), &user.LLMConfigRaw)
		if err != nil {
			return nil, fmt.Errorf("UnmarshalJSON failed: %v", err)
		}
	}
	normalizePrivilegedUser(&user)

	return &user, nil
}

// GetUsers get 1000 users order by updatetime
func GetUsers() ([]User, error) {
	rows, err := DB.Query("SELECT id, user_id, llm_config, update_time FROM users order by update_time limit 10000")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.UserId, &user.LLMConfig, &user.UpdateTime); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	// check error
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserLLMConfig update user llm config
func UpdateUserLLMConfig(userId string, llmConfig string) error {
	updateSQL := `UPDATE users SET llm_config = ?, update_time = ? WHERE user_id = ?`
	_, err := DB.Exec(updateSQL, llmConfig, time.Now().Unix(), userId)
	return err
}

// AddAvailToken add token
func AddAvailToken(userId string, token int) error {
	updateSQL := `UPDATE users SET avail_token = avail_token + ?, update_time = ? WHERE user_id = ?`
	_, err := DB.Exec(updateSQL, token, time.Now().Unix(), userId)
	return err
}

func AddToken(userId string, token int) error {
	updateSQL := `UPDATE users SET token = token + ?, update_time = ? WHERE user_id = ?`
	_, err := DB.Exec(updateSQL, token, time.Now().Unix(), userId)
	return err
}

func GetUserByPage(page, pageSize int, userId string) ([]User, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize

	// 构建 SQL
	var (
		whereSQL string
		args     []interface{}
	)

	if userId != "" {
		whereSQL = "WHERE user_id = ?"
		args = append(args, userId)
	}

	// 查询数据
	listSQL := fmt.Sprintf(`
		SELECT id, user_id, llm_config, token, update_time, avail_token, create_time
		FROM users %s
		ORDER BY id DESC
		LIMIT ? OFFSET ?`, whereSQL)
	args = append(args, pageSize, offset)

	rows, err := DB.Query(listSQL, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.UserId, &u.LLMConfig, &u.Token, &u.UpdateTime, &u.AvailToken, &u.CreateTime); err != nil {
			return nil, err
		}
		if u.LLMConfig != "" {
			err := json.Unmarshal([]byte(u.LLMConfig), &u.LLMConfigRaw)
			if err != nil {
				return nil, fmt.Errorf("UnmarshalJSON failed: %v", err)
			}
		}
		normalizePrivilegedUser(&u)
		users = append(users, u)
	}

	return users, nil
}

func GetUserCount(userId string) (int, error) {
	var whereSQL string
	args := make([]interface{}, 0)

	if userId != "" {
		whereSQL = "WHERE user_id = ?"
		args = append(args, userId)
	}

	// 查询总数
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereSQL)
	var total int
	if err := DB.QueryRow(countSQL, args...).Scan(&total); err != nil {
		return 0, err
	}

	return total, nil
}

func GetUserQuotaStats(page, pageSize int, userId, sortBy string) (*UserQuotaStats, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	users, err := getUsersForQuotaStats(userId)
	if err != nil {
		return nil, err
	}

	metrics := make([]UserQuotaMetric, 0, len(users))
	summary := UserQuotaSummary{}
	for _, user := range users {
		remaining := user.AvailToken - user.Token
		totalQuota := user.AvailToken
		usageRate := 0.0
		if user.Unlimited {
			remaining = -1
			totalQuota = -1
		} else if totalQuota > 0 {
			usageRate = float64(user.Token) / float64(totalQuota)
		}
		metrics = append(metrics, UserQuotaMetric{
			ID:             user.ID,
			UserID:         user.UserId,
			Token:          user.Token,
			AvailToken:     totalQuota,
			RemainingToken: remaining,
			UsageRate:      usageRate,
			Unlimited:      user.Unlimited,
			CreateTime:     user.CreateTime,
			UpdateTime:     user.UpdateTime,
		})
		summary.TotalUsers++
		summary.TotalUsedToken += user.Token
		if user.Unlimited {
			summary.UnlimitedUsers++
			continue
		}
		summary.TotalRemainToken += remaining
		summary.TotalQuotaToken += totalQuota
	}
	if summary.TotalQuotaToken > 0 {
		summary.AverageUsageRate = float64(summary.TotalUsedToken) / float64(max(summary.TotalQuotaToken, 1))
	}

	sortedMetrics := append([]UserQuotaMetric(nil), metrics...)
	normalizedSortBy := normalizeQuotaSort(sortBy)
	sortUserQuotaMetrics(sortedMetrics, normalizedSortBy)

	topUsed := append([]UserQuotaMetric(nil), metrics...)
	sortUserQuotaMetrics(topUsed, "high_used")
	if len(topUsed) > 5 {
		topUsed = topUsed[:5]
	}

	lowestRemaining := append([]UserQuotaMetric(nil), metrics...)
	sortUserQuotaMetrics(lowestRemaining, "low_remaining")
	if len(lowestRemaining) > 5 {
		lowestRemaining = lowestRemaining[:5]
	}

	start := (page - 1) * pageSize
	if start > len(sortedMetrics) {
		start = len(sortedMetrics)
	}
	end := start + pageSize
	if end > len(sortedMetrics) {
		end = len(sortedMetrics)
	}

	return &UserQuotaStats{
		Summary:         summary,
		TopUsed:         topUsed,
		LowestRemaining: lowestRemaining,
		Distribution:    buildQuotaDistribution(metrics),
		List:            sortedMetrics[start:end],
		Total:           len(sortedMetrics),
		Page:            page,
		PageSize:        pageSize,
		SortBy:          normalizedSortBy,
		UserID:          userId,
	}, nil
}

func getUsersForQuotaStats(userId string) ([]User, error) {
	var (
		whereSQL string
		args     []interface{}
	)
	if userId != "" {
		whereSQL = "WHERE user_id = ?"
		args = append(args, userId)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, llm_config, token, avail_token, update_time, create_time
		FROM users %s
	`, whereSQL)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.UserId, &user.LLMConfig, &user.Token, &user.AvailToken, &user.UpdateTime, &user.CreateTime); err != nil {
			return nil, err
		}
		normalizePrivilegedUser(&user)
		users = append(users, user)
	}
	return users, rows.Err()
}

func normalizeQuotaSort(sortBy string) string {
	switch sortBy {
	case "low_remaining", "usage_rate", "latest", "user_id":
		return sortBy
	case "high_used":
		return sortBy
	default:
		return "high_used"
	}
}

func sortUserQuotaMetrics(items []UserQuotaMetric, sortBy string) {
	switch sortBy {
	case "low_remaining":
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Unlimited != items[j].Unlimited {
				return !items[i].Unlimited
			}
			if items[i].RemainingToken == items[j].RemainingToken {
				return items[i].Token > items[j].Token
			}
			return items[i].RemainingToken < items[j].RemainingToken
		})
	case "usage_rate":
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Unlimited != items[j].Unlimited {
				return !items[i].Unlimited
			}
			if items[i].UsageRate == items[j].UsageRate {
				return items[i].Token > items[j].Token
			}
			return items[i].UsageRate > items[j].UsageRate
		})
	case "latest":
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].UpdateTime == items[j].UpdateTime {
				return items[i].ID > items[j].ID
			}
			return items[i].UpdateTime > items[j].UpdateTime
		})
	case "user_id":
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].UserID < items[j].UserID
		})
	default:
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].Token == items[j].Token {
				return items[i].UsageRate > items[j].UsageRate
			}
			return items[i].Token > items[j].Token
		})
	}
}

func buildQuotaDistribution(items []UserQuotaMetric) []UserQuotaBucket {
	buckets := []UserQuotaBucket{
		{Label: "0-25%", Min: 0, Max: 25},
		{Label: "25-50%", Min: 25, Max: 50},
		{Label: "50-75%", Min: 50, Max: 75},
		{Label: "75-90%", Min: 75, Max: 90},
		{Label: "90-100%", Min: 90, Max: 100},
		{Label: "100%+", Min: 100, Max: 101},
		{Label: "∞", Min: -1, Max: -1},
	}

	for _, item := range items {
		if item.Unlimited {
			buckets[6].Count++
			continue
		}
		percent := int(item.UsageRate * 100)
		switch {
		case percent < 25:
			buckets[0].Count++
		case percent < 50:
			buckets[1].Count++
		case percent < 75:
			buckets[2].Count++
		case percent < 90:
			buckets[3].Count++
		case percent < 100:
			buckets[4].Count++
		default:
			buckets[5].Count++
		}
	}
	return buckets
}

func normalizePrivilegedUser(user *User) {
	if user == nil || !conf.IsPrivilegedUser(user.UserId) {
		return
	}
	user.Unlimited = true
	user.AvailToken = -1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func GetDailyNewUsers(days int) ([]DailyStat, error) {
	var query string
	var intervalSeconds int64

	if days <= 3 {
		intervalSeconds = 3600 // 每小时
	} else if days <= 7 {
		intervalSeconds = 3 * 3600 // 每3小时
	} else {
		intervalSeconds = 86400 // 每天
	}

	if conf.BaseConfInfo.DBType == "mysql" {
		query = `
			SELECT
				FLOOR(create_time / ?) * ? AS time_group,
				COUNT(DISTINCT user_id) AS new_count
			FROM users
			WHERE create_time >= UNIX_TIMESTAMP(DATE_SUB(NOW(), INTERVAL ? DAY))
			GROUP BY time_group
			ORDER BY time_group DESC;
		`
	} else if conf.BaseConfInfo.DBType == "sqlite3" {
		query = `
			SELECT
				(create_time / ?) * ? AS time_group,
				COUNT(DISTINCT user_id) AS new_count
			FROM users
			WHERE create_time >= strftime('%s', date('now', ? || ' days'))
			GROUP BY time_group
			ORDER BY time_group DESC;
		`
	} else {
		return nil, fmt.Errorf("unsupported DBType: %s", conf.BaseConfInfo.DBType)
	}

	var rows *sql.Rows
	var err error
	if conf.BaseConfInfo.DBType == "sqlite3" {
		rows, err = DB.Query(query, intervalSeconds, intervalSeconds, -days)
	} else {
		rows, err = DB.Query(query, intervalSeconds, intervalSeconds, days)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []DailyStat
	for rows.Next() {
		var stat DailyStat
		if err := rows.Scan(&stat.Date, &stat.NewCount); err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func GetCtxUserInfo(ctx context.Context) *User {
	userInfo, ok := ctx.Value("user_info").(*User)
	if ok {
		return userInfo
	}

	return nil
}

func DeleteUserByUserID(ctx context.Context, userId string) error {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().Unix()

	recordUpdateResult, err := tx.Exec(
		`UPDATE records SET is_deleted = 1, update_time = ? WHERE user_id = ? AND is_deleted = 0`,
		now,
		userId,
	)
	if err != nil {
		return err
	}

	userDeleteResult, err := tx.Exec(`DELETE FROM users WHERE user_id = ?`, userId)
	if err != nil {
		return err
	}

	userRowsAffected, err := userDeleteResult.RowsAffected()
	if err != nil {
		return err
	}

	recordRowsAffected, err := recordUpdateResult.RowsAffected()
	if err != nil {
		return err
	}

	if userRowsAffected == 0 && recordRowsAffected == 0 {
		return sql.ErrNoRows
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	MsgRecord.Delete(userId)
	return nil
}
