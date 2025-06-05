package main

import (
	"fmt"
	"time"

	"github.com/saisona/simba"
	"github.com/slack-go/slack"
	"gorm.io/gorm"
)

type homeViewAdminInfo struct {
	Total       float64
	TotalByUser map[string]float64
	Coworkers   []*simba.User
}

func NewHomeViewAdminInfo(dbClient *gorm.DB) (*homeViewAdminInfo, error) {
	hvi := &homeViewAdminInfo{
		Coworkers:   []*simba.User{},
		TotalByUser: make(map[string]float64),
	}
	if err := hvi.fetchWeeklyMoodsByUser(dbClient); err != nil {
		return nil, err
	}
	return hvi, nil
}

func (hvi *homeViewAdminInfo) fetchWeeklyMoodsByUser(dbClient *gorm.DB) error {
	var coworkers []*simba.User

	if tx := dbClient.Debug().Find(&coworkers); tx.Error != nil {
		return tx.Error
	}

	for _, u := range coworkers {
		var wm []simba.DailyMood
		before := time.Now().Add(-time.Hour * 24 * 14)
		after := time.Now()
		if tx := dbClient.Debug().Where("user_id=?", u.ID).Where("created_at between ? AND ?", before, after).Limit(14).Order("created_at DESC").Find(&wm); tx.Error != nil {
			return tx.Error
		}
		u.Moods = wm
		hvi.TotalByUser[u.Username] = float64(len(wm))
		hvi.Total += float64(len(wm))
	}

	hvi.Coworkers = coworkers
	return nil
}

func (hvi homeViewAdminInfo) mapAllCount() map[string]int {
	var moodCountMap map[string]int = map[string]int{}
	for _, i := range hvi.Coworkers {
		for _, s := range i.Moods {
			moodCountMap[s.Mood] += 1
		}
	}
	return moodCountMap
}

func (hvai homeViewAdminInfo) avgTotal(m map[string]int) map[string]float64 {
	avg := make(map[string]float64)
	for k := range m {
		avg[k] = (float64(m[k]) / hvai.Total) * 100
	}
	return avg
}

func (hvai homeViewAdminInfo) avgByUser(
	avgUser map[string]map[string]int,
) map[string]map[string]float64 {
	avg := make(map[string]map[string]float64)
	for u, m := range avgUser {
		avg[u] = make(map[string]float64)
		for k, v := range m {
			avg[u][k] = (float64(v) / hvai.TotalByUser[u]) * 100
		}
	}
	return avg
}

func (hvi homeViewAdminInfo) mapByUserCount() map[string]map[string]int {
	var moodCountMap map[string]map[string]int = make(map[string]map[string]int, len(hvi.Coworkers))
	for _, k := range hvi.Coworkers {
		if len(k.Moods) > 0 {
			moodCountMap[k.Username] = make(map[string]int, len(k.Moods))
			for _, m := range k.Moods {
				moodCountMap[k.Username][m.Mood] += 1
			}
		} else {
			continue
		}
	}
	return moodCountMap
}

// @desc Render Home view admin or update depending on slackChannelId is given or not
// @params user is a DB representation of a Simba user
// @params [slackChannelId] is optionnal given if already known or not used for update
// @returns Blocks to be send to update Simba Home view
func handleAppHomeViewAdmin(
	user *simba.User,
	config *simba.Config,
	dbClient *gorm.DB,
) slack.Blocks {
	basicText := slackTextBlock("Simba Application (Admin)")
	slackHeaderBlock := slack.NewHeaderBlock(basicText)

	hvai, err := NewHomeViewAdminInfo(dbClient)
	if err != nil {
		panic(err)
	}

	slackAvgTotalTitleInfo := slack.NewHeaderBlock(slackTextBlock("Week informations"))

	blockSet := []slack.Block{slackHeaderBlock, slack.NewDividerBlock(), slackAvgTotalTitleInfo}
	for u, a := range hvai.avgTotal(hvai.mapAllCount()) {
		text := fmt.Sprintf("%s %.2f%%", simba.FromMoodToSmiley(u), a)
		buttonBlock := slack.NewButtonBlockElement("_", "", slackTextBlock(text))
		actionBlock := slack.NewActionBlock(
			fmt.Sprintf("total_%s_%d", u, time.Now().Unix()),
			buttonBlock,
		)
		blockSet = append(blockSet, actionBlock)
	}

	for u, m := range hvai.avgByUser(hvai.mapByUserCount()) {
		slackAvgByUserSectionTitle := slack.NewHeaderBlock(slackTextBlock(u))
		blockSet = append(blockSet, slackAvgByUserSectionTitle)
		elemBlock := []slack.BlockElement{}
		for k, v := range m {
			text := fmt.Sprintf("%s %.2f%%", simba.FromMoodToSmiley(k), v)
			buttonBlock := slack.NewButtonBlockElement(
				fmt.Sprintf("%s_%d", k, time.Now().Unix()),
				"",
				slackTextBlock(text),
			)
			elemBlock = append(elemBlock, buttonBlock)
		}
		actionBlock := slack.NewActionBlock(
			fmt.Sprintf("action_block_user_%s_%d", u, time.Now().Unix()+1),
			elemBlock...)
		blockSet = append(blockSet, actionBlock, slack.NewDividerBlock())
	}

	return slack.Blocks{
		BlockSet: blockSet,
	}
}
